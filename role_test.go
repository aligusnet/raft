package raft

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestRole(t *testing.T) {
	Convey("ExitRole should just return ExitRoleHandle", t, func() {
		So(exitRoleInstance.RunRole(), ShouldEqual, ExitRoleHandle)
	})
}
