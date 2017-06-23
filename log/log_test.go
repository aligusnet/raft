package log

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestLog(t *testing.T) {
	Convey("Given initialized Log", t, func() {
		l := New()

		So(l.Size(), ShouldEqual, 0)

		Convey("The Log should be able to append commands", func() {
			size := l.Size()
			index := l.Append(10, []byte("Command1"))
			So(l.Size(), ShouldEqual, size+1)
			So(l.Get(index).Command, ShouldResemble, []byte("Command1"))
			So(l.Get(index).Term, ShouldEqual, 10)
		})

		Convey("The Log should be able to erase last commands", func() {
			l.Append(0, []byte("Command1"))
			index := l.Append(0, []byte("Command2"))
			l.Append(0, []byte("Command3"))
			l.Append(0, []byte("Command4"))

			l.EraseAfter(index)
			So(l.Size(), ShouldEqual, index+1)
			So(l.Get(index).Command, ShouldResemble, []byte("Command2"))
		})
	})
}
