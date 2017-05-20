package raft

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestLog(t *testing.T) {
	Convey("Given initialized log", t, func() {
		log := NewLog()

		So(log.Size(), ShouldEqual, 0)

		Convey("The log should be able to append commands", func() {
			size := log.Size()
			index := log.Append(10, []byte("Command1"))
			So(log.Size(), ShouldEqual, size+1)
			So(log.Get(index).Command, ShouldResemble, []byte("Command1"))
			So(log.Get(index).Term, ShouldEqual, 10)
		})

		Convey("The log should be able to erase last commands", func() {
			log.Append(0, []byte("Command1"))
			index := log.Append(0, []byte("Command2"))
			log.Append(0, []byte("Command3"))
			log.Append(0, []byte("Command4"))

			log.EraseAfter(index)
			So(log.Size(), ShouldEqual, index+1)
			So(log.Get(index).Command, ShouldResemble, []byte("Command2"))
		})
	})
}
