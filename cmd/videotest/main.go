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

package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/golang/glog"
	"github.com/ssmall/kubetranscode/pkg/video"
)

func main() {
	flag.Parse()

	ctx := context.Background()

	out, err := video.Transcode(ctx, "/home/spencer/Downloads/nocco.mp4", 5*time.Second, 2*time.Minute)

	if err != nil {
		log.Fatal(err)
	}

	total := 0
	for next, more := <-out; more; next, more = <-out {
		total += len(next)
	}
	glog.Infoln("Total transcoded bytes:", total)
}
