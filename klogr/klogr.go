// Package klogr implements github.com/go-logr/logr.Logger in terms of
// k8s.io/klog.
package klogr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"runtime"
	"sort"
	"strings"

	"github.com/go-logr/logr"
	"github.com/jsamunderu/klog/v2"
)

// New returns a logr.Logger which is implemented by klog.
func New() logr.Logger {
	return klogger{
		level:  0,
		prefix: "",
		values: nil,
	}
}

type klogger struct {
	level  int
	prefix string
	values []interface{}
}

func (l klogger) clone() klogger {
	return klogger{
		level:  l.level,
		prefix: l.prefix,
		values: copySlice(l.values),
	}
}

func copySlice(in []interface{}) []interface{} {
	out := make([]interface{}, len(in))
	copy(out, in)
	return out
}

// Magic string for intermediate frames that we should ignore.
const autogeneratedFrameName = "<autogenerated>"

// Discover how many frames we need to climb to find the caller. This approach
// was suggested by Ian Lance Taylor of the Go team, so it *should* be safe
// enough (famous last words).
func framesToCaller() int {
	// 1 is the immediate caller.  3 should be too many.
	for i := 1; i < 3; i++ {
		_, file, _, _ := runtime.Caller(i + 1) // +1 for this function's frame
		if file != autogeneratedFrameName {
			return i
		}
	}
	return 1 // something went wrong, this is safe
}

// trimDuplicates will deduplicates elements provided in multiple KV tuple
// slices, whilst maintaining the distinction between where the items are
// contained.
func trimDuplicates(kvLists ...[]interface{}) [][]interface{} {
	// maintain a map of all seen keys
	seenKeys := map[interface{}]struct{}{}
	// build the same number of output slices as inputs
	outs := make([][]interface{}, len(kvLists))
	// iterate over the input slices backwards, as 'later' kv specifications
	// of the same key will take precedence over earlier ones
	for i := len(kvLists) - 1; i >= 0; i-- {
		// initialise this output slice
		outs[i] = []interface{}{}
		// obtain a reference to the kvList we are processing
		kvList := kvLists[i]

		// start iterating at len(kvList) - 2 (i.e. the 2nd last item) for
		// slices that have an even number of elements.
		// We add (len(kvList) % 2) here to handle the case where there is an
		// odd number of elements in a kvList.
		// If there is an odd number, then the last element in the slice will
		// have the value 'null'.
		for i2 := len(kvList) - 2 + (len(kvList) % 2); i2 >= 0; i2 -= 2 {
			k := kvList[i2]
			// if we have already seen this key, do not include it again
			if _, ok := seenKeys[k]; ok {
				continue
			}
			// make a note that we've observed a new key
			seenKeys[k] = struct{}{}
			// attempt to obtain the value of the key
			var v interface{}
			// i2+1 should only ever be out of bounds if we handling the first
			// iteration over a slice with an odd number of elements
			if i2+1 < len(kvList) {
				v = kvList[i2+1]
			}
			// add this KV tuple to the *start* of the output list to maintain
			// the original order as we are iterating over the slice backwards
			outs[i] = append([]interface{}{k, v}, outs[i]...)
		}
	}
	return outs
}

func flatten(kvList ...interface{}) string {
	keys := make([]string, 0, len(kvList))
	vals := make(map[string]interface{}, len(kvList))
	for i := 0; i < len(kvList); i += 2 {
		k, ok := kvList[i].(string)
		if !ok {
			panic(fmt.Sprintf("key is not a string: %s", pretty(kvList[i])))
		}
		var v interface{}
		if i+1 < len(kvList) {
			v = kvList[i+1]
		}
		keys = append(keys, k)
		vals[k] = v
	}
	sort.Strings(keys)
	buf := bytes.Buffer{}
	for i, k := range keys {
		v := vals[k]
		if i > 0 {
			buf.WriteRune(' ')
		}
		buf.WriteString(pretty(k))
		buf.WriteString("=")
		buf.WriteString(pretty(v))
	}
	return buf.String()
}

func pretty(value interface{}) string {
	if err, ok := value.(error); ok {
		if _, ok := value.(json.Marshaler); !ok {
			value = err.Error()
		}
	}
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	encoder.Encode(value)
	return strings.TrimSpace(string(buffer.Bytes()))
}

func (l klogger) Info(msg string, kvList ...interface{}) {
	if l.Enabled() {
		msgStr := flatten("msg", msg)
		trimmed := trimDuplicates(l.values, kvList)
		fixedStr := flatten(trimmed[0]...)
		userStr := flatten(trimmed[1]...)
		klog.InfoDepth(framesToCaller(), l.prefix, " ", msgStr, " ", fixedStr, " ", userStr)
	}
}

func (l klogger) Enabled() bool {
	return bool(klog.V(klog.Level(l.level)).Enabled())
}

func (l klogger) Error(err error, msg string, kvList ...interface{}) {
	msgStr := flatten("msg", msg)
	var loggableErr interface{}
	if err != nil {
		loggableErr = err.Error()
	}
	errStr := flatten("error", loggableErr)
	trimmed := trimDuplicates(l.values, kvList)
	fixedStr := flatten(trimmed[0]...)
	userStr := flatten(trimmed[1]...)
	klog.ErrorDepth(framesToCaller(), l.prefix, " ", msgStr, " ", errStr, " ", fixedStr, " ", userStr)
}

func (l klogger) V(level int) logr.InfoLogger {
	new := l.clone()
	new.level = level
	return new
}

// WithName returns a new logr.Logger with the specified name appended.  klogr
// uses '/' characters to separate name elements.  Callers should not pass '/'
// in the provided name string, but this library does not actually enforce that.
func (l klogger) WithName(name string) logr.Logger {
	new := l.clone()
	if len(l.prefix) > 0 {
		new.prefix = l.prefix + "/"
	}
	new.prefix += name
	return new
}

func (l klogger) WithValues(kvList ...interface{}) logr.Logger {
	new := l.clone()
	new.values = append(new.values, kvList...)
	return new
}

var _ logr.Logger = klogger{}
var _ logr.InfoLogger = klogger{}
