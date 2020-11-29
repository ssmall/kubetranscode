// Copyright 2020 Spencer Small
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package video provides methods to clip and transcode video using ffmpeg
package video

import (
	"context"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/golang/glog"
	"golang.org/x/sync/errgroup"
)

const bufferSize = 56 * 1024 // 56KiB

// Transcode takes an input video and transcodes the section between start and end
func Transcode(ctx context.Context, filename string, start time.Duration, end time.Duration) (<-chan []byte, error) {
	g, gCtx := errgroup.WithContext(ctx)
	cmd := exec.CommandContext(gCtx, "ffmpeg",
		"-ss", formatHHMMSS(start),
		"-i", filename,
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-tune", "zerolatency",
		"-profile:v", "high",
		"-level:v", "3.1",
		"-movflags", "+faststart",
		"-c:a", "aac",
		"-b:a", "192k",
		"-ac", "2",
		"-f", "mpegts",
		"-to", formatHHMMSS(end),
		"-copyts",
		"-")

	glog.V(1).Infof("Executing: %s", cmd)

	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	var stdErr io.ReadCloser
	if glog.V(2) {
		stdErr, err = cmd.StderrPipe()
		if err != nil {
			return nil, err
		}
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	if stdErr != nil {
		go io.Copy(os.Stderr, stdErr)
	}

	buf := make(chan []byte, 100)

	g.Go(func() error {
		b := make([]byte, bufferSize)
		n, err := out.Read(b)
		for err == nil {
			select {
			case <-gCtx.Done():
				glog.V(1).Infoln("Cancelling write due to context cancellation")
				return nil
			default:
				buf <- b[:n]
				b = make([]byte, bufferSize)
				n, err = out.Read(b)
			}
		}

		if err == io.EOF {
			return nil
		}
		return err
	})

	g.Go(func() error {
		err := cmd.Wait()
		switch v := err.(type) {
		case *exec.ExitError:
			glog.V(1).Infoln("Command failed with output:", v)
		case nil:
			glog.V(1).Infoln("Transcoding completed succesfully")
		default:
			glog.V(1).Infoln("Command failed:", v)
		}
		return err
	})

	go func() {
		defer close(buf)
		err = g.Wait()

		if err != nil {
			glog.Errorln("Error processing stream:", err)
		}
	}()

	return buf, nil
}
