package smplt_test

import (
	"testing"

	. "github.com/meatballhat/box-o-sand/gotime/smplt"
)

const BASIC_RENDERED_TXT_SIMPLATE = `
import (
    "time"
)

type Dance struct {
    Who  string
    When time.Time
}

d := &Dance{
    Who: "Everybody",
    When: time.Now(),
}

{{d.Who}} Dance {{d.When}}!
`

const BASIC_STATIC_TXT_SIMPLATE = `
Everybody Dance Now!
`

func TestDetectsRenderedSimplate(t *testing.T) {
	s := SimplateFromString("basic-rendered.txt", BASIC_RENDERED_TXT_SIMPLATE)
	if s.Type != SIMPLATE_TYPE_RENDERED {
		t.Errorf("Simplate detected as %s instead of %s", s.Type, SIMPLATE_TYPE_RENDERED)
	}
}

func TestDetectsStaticSimplate(t *testing.T) {
	s := SimplateFromString("basic-static.txt", BASIC_STATIC_TXT_SIMPLATE)
	if s.Type != SIMPLATE_TYPE_STATIC {
		t.Errorf("Simplate detected as %s instead of %s", s.Type, SIMPLATE_TYPE_STATIC)
	}
}
