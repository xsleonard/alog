package alog

import (
	"errors"
	stdlog "log"
	"testing"

	"gopkg.in/check.v1"
)

func Test(t *testing.T) { check.TestingT(t) }

type Suite struct{}

var _ = check.Suite(&Suite{})

func (s *Suite) TestMeta(c *check.C) {
	m := &Meta{}

	// Not set
	c.Assert(m.get("foo"), check.IsNil)

	// Set, get
	m.set("foo", "bar")
	c.Assert(m.get("foo"), check.Equals, "bar")

	// Del
	m.del("foo")
	c.Assert(m.get("foo"), check.IsNil)

	m.set("foo", "bar")
	m.set("t", 7)
	c.Assert(m.entries["foo"].order, check.Equals, 0)
	c.Assert(m.entries["t"].order, check.Equals, 1)

	// Format
	f := m.format(", ", "*%s*")
	c.Assert(f, check.Equals, "*foo=bar, t=7*")

	// Overwrite, preserves order
	m.set("foo", "baz")
	c.Assert(m.entries["foo"].order, check.Equals, 0)
	f = m.format(", ", "*%s*")
	c.Assert(f, check.Equals, "*foo=baz, t=7*")

	// Copy
	n := m.copy()
	n.set("foo", "bar")
	c.Assert(m.get("foo"), check.Equals, "baz")
	c.Assert(n.get("foo"), check.Equals, "bar")
}

type Thief struct {
	msgs []string
}

func (t *Thief) Write(s []byte) (int, error) {
	t.msgs = append(t.msgs, string(s))
	return len(s), nil
}

func (t *Thief) last() string {
	return t.msgs[len(t.msgs)-1]
}

func checkLast(c *check.C, t *Thief, expected string) {
	c.Assert(t.last(), check.Equals, expected+"\n")
}

func (s *Suite) TestLogNilSafe(c *check.C) {
	var log *Log

	c.Assert(log.Copy(), check.IsNil)
	c.Assert(log.Set("foo", "bar"), check.IsNil)
	c.Assert(log.SetError(errors.New("foo")), check.IsNil)

	c.Assert(log.With("foo", "bar"), check.IsNil)
	c.Assert(log.WithError(errors.New("foo")), check.IsNil)

	c.Assert(log.Sprint("foo", "bar"), check.Equals, "foobar")
	c.Assert(log.Sprintf("%s %d", "foo", 7), check.Equals, "foo 7")
	c.Assert(log.Sprintln("foo", "bar"), check.Equals, "foo bar\n")

	c.Assert(func() { log.Panic("foo") }, check.Panics, "foo")
	c.Assert(func() { log.Panicf("foo") }, check.Panics, "foo")
	c.Assert(func() { log.Panicln("foo") }, check.Panics, "foo\n")

	log.Print("foo")
	log.Printf("foo")
	log.Println("foo")

	// Fatal* cannot be checked

	c.Assert(log, check.IsNil)
}

func (s *Suite) TestLog(c *check.C) {
	t := &Thief{}
	log := New(t)

	// Disable timestamps
	log.SetFlags(0)

	// Basic settings
	log.Print("foo")
	checkLast(c, t, "foo")

	// Set meta
	log.Set("foo", "bar")
	log.Set("key", 7)
	log.Print("test")
	checkLast(c, t, "[foo=bar key=7] test")

	// With meta
	log2 := log.With("foo", "baz")
	log2.Print("test")
	checkLast(c, t, "[foo=baz key=7] test")
	// Original is unchanged
	log.Print("test")
	checkLast(c, t, "[foo=bar key=7] test")

	// Set error, error is quoted
	log.SetError(errors.New("bad"))
	log.Print("test")
	checkLast(c, t, "[foo=bar key=7 error='bad'] test")

	// With error
	log2 = log.WithError(errors.New("good"))
	log2.Print("test")
	checkLast(c, t, "[foo=bar key=7 error='good'] test")
	log.Print("test")
	checkLast(c, t, "[foo=bar key=7 error='bad'] test")

	// Panic string shouldn't include the prefix
	c.Assert(func() { log.Panic("xxx") }, check.Panics, "xxx")

	// Check with a timestamp
	log = New(t)
	log.SetFlags(stdlog.LstdFlags)
	log.Print("foo")
	c.Assert(t.last(), check.Matches, `\d{4}/\d\d/\d\d \d\d:\d\d:\d\d foo\n`)
}
