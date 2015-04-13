package alog

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

const defaultCalldepth = 3

// Logger used when *alog.Log is nil
var fallback = log.New(os.Stdout, "", log.Flags())

// Custom logger that prints the prefix after the timestamp instead of before it
type Log struct {
	*log.Logger
	*Meta
	out       io.Writer
	calldepth int
}

func New(out io.Writer) *Log {
	return newAdvanced(out, log.Flags(), defaultCalldepth)
}

func newAdvanced(out io.Writer, flags, calldepth int) *Log {
	return &Log{log.New(out, "", flags), &Meta{}, out, calldepth}
}

func (a *Log) Copy() *Log {
	if a == nil {
		return nil
	}
	return &Log{log.New(a.out, "", a.Logger.Flags()), a.Meta.copy(), a.out, defaultCalldepth}
}

// Sets a key-value for inclusion in the log prefix
func (a *Log) Set(k string, v interface{}) *Log {
	if a == nil {
		return nil
	}
	a.Meta.set(k, v)
	return a
}

func (a *Log) SetError(err error) *Log {
	return a.Set("error", fmt.Sprintf("'%s'", err))
}

// Shorthand for .Copy().Set(k, v).  Use for temporary k:v values.
func (a *Log) With(k string, v interface{}) *Log {
	return a.Copy().Set(k, v)
}

// Shorthand for .With("error", err),
// and encapsulates the error string in single quotes.
func (a *Log) WithError(err error) *Log {
	return a.Copy().SetError(err)
}

func (a *Log) Sprint(v ...interface{}) string {
	// Sprint doesn't put spaces between the arguments, add one for the prefix
	return fmt.Sprint(append(a.buildPrefixArgs(), v...)...)
}

func (a *Log) Sprintf(f string, v ...interface{}) string {
	return fmt.Sprintf(a.addPrefix(f), v...)
}

func (a *Log) Sprintln(v ...interface{}) string {
	return fmt.Sprintln(append(a.buildPrefixArgs(), v...)...)
}

func (a *Log) Fatal(v ...interface{}) {
	a.output(a.Sprint(v...))
	os.Exit(1)
}

func (a *Log) Fatalf(f string, v ...interface{}) {
	a.output(a.Sprintf(f, v...))
	os.Exit(1)
}

func (a *Log) Fatalln(v ...interface{}) {
	a.output(a.Sprintln(v...))
	os.Exit(1)
}

func (a *Log) Panic(v ...interface{}) {
	a.output(a.Sprint(v...))
	panic(fmt.Sprint(v...))
}

func (a *Log) Panicf(f string, v ...interface{}) {
	a.output(a.Sprintf(f, v...))
	panic(fmt.Sprintf(f, v...))
}

func (a *Log) Panicln(v ...interface{}) {
	a.output(a.Sprintln(v...))
	panic(fmt.Sprintln(v...))
}

func (a *Log) Print(v ...interface{}) {
	a.output(a.Sprint(v...))
}

func (a *Log) Printf(f string, v ...interface{}) {
	a.output(a.Sprintf(f, v...))
}

func (a *Log) Println(v ...interface{}) {
	a.output(a.Sprintln(v...))
}

func (a *Log) output(s string) {
	if a == nil {
		fallback.Output(defaultCalldepth, s)
	} else {
		a.Logger.Output(a.calldepth, s)
	}
}

func (a *Log) prefix() string {
	if a == nil {
		return ""
	}
	return a.Meta.format(" ", "[%s]")
}

// Prepends the prefix to the output string
func (a *Log) addPrefix(s string) string {
	prefix := a.prefix()
	if prefix != "" {
		return prefix + " " + s
	}
	return s
}

// Returns the prefix as interface args for prepending in a Print() call
func (a *Log) buildPrefixArgs() []interface{} {
	prefix := a.prefix()
	if prefix != "" {
		return []interface{}{prefix, " "}
	}
	return nil
}

////////////////////////////////////////////////
//// Meta object for managing stored values ////
////////////////////////////////////////////////

type MetaEntry struct {
	value interface{}
	order int
}

type Meta struct {
	entries map[string]MetaEntry
	mutex   sync.RWMutex
}

func (m *Meta) get(k string) interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.entries[k].value
}

func (m *Meta) set(k string, v interface{}) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.entries == nil {
		m.entries = make(map[string]MetaEntry)
	}

	vi, ok := m.entries[k]
	if ok {
		m.entries[k] = MetaEntry{v, vi.order}
	} else {
		m.entries[k] = MetaEntry{v, len(m.entries)}
	}
}

func (m *Meta) del(k string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.entries, k)
}

func (m *Meta) format(delim, format string) string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if len(m.entries) == 0 {
		return ""
	}

	pts := make([]string, len(m.entries))
	for k, vi := range m.entries {
		pts[vi.order] = fmt.Sprintf("%s=%+v", k, vi.value)
	}

	s := strings.Join(pts, delim)
	if format == "" {
		return s
	}
	return fmt.Sprintf(format, s)
}

func (m *Meta) copy() *Meta {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var entries map[string]MetaEntry
	if m.entries != nil {
		entries = make(map[string]MetaEntry, len(m.entries))
		for k, v := range m.entries {
			entries[k] = v
		}
	}

	return &Meta{entries, sync.RWMutex{}}
}
