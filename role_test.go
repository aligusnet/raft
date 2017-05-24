package raft

import (
	"context"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestRole(t *testing.T) {
	Convey("ExitRole should just return ExitRoleHandle", t, func() {
		role, _ := exitRoleInstance.RunRole(context.Background(), nil)
		So(role, ShouldEqual, ExitRoleHandle)
	})
}
