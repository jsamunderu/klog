package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/jsamunderu/klog"
)

func main() {
	klog.InitFlags(nil)
	flag.Set("logtostderr", "false")
	flag.Set("alsologtostderr", "false")
	flag.Set("skip_severity_headers", "true")
	flag.Set("skip_time_headers", "true")
	flag.Set("skip_pid_headers", "true")
	flag.Parse()

	buf := new(bytes.Buffer)
	klog.SetOutput(buf)
	klog.Info("nice to meet you")
	klog.Flush()

	fmt.Printf("LOGGED: %s", buf.String())
}
