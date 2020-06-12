package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/jsamunderu/klog/v2"
)

func main() {
	klog.InitFlags(nil)
	flag.Set("logtostderr", "false")
	flag.Set("alsologtostderr", "false")
	flag.Set("skip_severity_headers", "false")
	flag.Set("skip_time_headers", "false")
	flag.Set("skip_pid_headers", "false")
	flag.Set("skip_caller_function", "false")
	flag.Parse()

	buf := new(bytes.Buffer)
	klog.SetOutput(buf)
	//klog.Info("nice to meet you")
	//klog.InfoS("nice to meet you")
	klog.InfoS("nice to meet you", "key", "value")
	//klog.Warning("xxxx")
	klog.Info("nice to meet you")
	klog.Flush()

	fmt.Printf("LOGGED: %s", buf.String())
}
