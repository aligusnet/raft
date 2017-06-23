package server

import (
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/net/context"
	"testing"
)

func TestRole(t *testing.T) {
	Convey("ExitRole should just return ExitRoleHandle", t, func() {
		role, _ := exitRoleInstance.RunRole(context.Background(), nil)
		So(role, ShouldEqual, ExitRoleHandle)
	})
}
