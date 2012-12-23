package smplt_test

import (
	"mime"
	"testing"

	. "github.com/meatballhat/box-o-sand/gotime/smplt"
)

const (
	BASIC_RENDERED_TXT_SIMPLATE = `
import (
    "time"
)

type Dance struct {
    Who  string
    When time.Time
}

D := &Dance{
    Who:  "Everybody",
    When: time.Now(),
}

{{.D.Who}} Dance {{.D.When}}!
`
	BASIC_STATIC_TXT_SIMPLATE = `
Everybody Dance Now!
`
	BASIC_JSON_SIMPLATE = `
import (
    "time"
)

type Dance struct {
    Who  string
    When time.Time
}

D := &Dance{
    Who:  "Everybody",
    When: time.Now(),
}
response.SetBody(D)
`
	BASIC_NEGOTIATED_SIMPLATE = `
import (
    "time"
)

type Dance struct {
    Who  string
    When time.Time
}

D := &Dance{
    Who:  "Everybody",
    When: time.Now(),
}
 text/plain
{{.D.Who}} Dance {{.D.When}}!

 application/json
{"who":"{{.D.Who}}","when":"{{.D.When}}"}
`
)

func TestSimplateKnowsItsFilename(t *testing.T) {
	s := SimplateFromString("hasty-decisions.txt", "herpherpderpherp")
	if s.Filename != "hasty-decisions.txt" {
		t.Errorf("Simplate filename incorrectly assigned as %s instead of %s",
			s.Filename, "hasty-decisions.txt")
	}
}

func TestSimplateKnowsItsContentType(t *testing.T) {
	s := SimplateFromString("hasty-decisions.js", "function herp() { return 'derp'; }")
	expected := mime.TypeByExtension(".js")

	if s.ContentType != expected {
		t.Errorf("Simplate content type incorrectly assigned as %s instead of %s",
			s.ContentType, expected)
	}
}

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

func TestDetectsJSONSimplates(t *testing.T) {
	s := SimplateFromString("basic.json", BASIC_JSON_SIMPLATE)
	if s.Type != SIMPLATE_TYPE_JSON {
		t.Errorf("Simplate detected as %s instead of %s", s.Type, SIMPLATE_TYPE_JSON)
	}
}

func TestDetectsNegotiatedSimplates(t *testing.T) {
	s := SimplateFromString("hork", BASIC_NEGOTIATED_SIMPLATE)
	if s.Type != SIMPLATE_TYPE_NEGOTIATED {
		t.Errorf("Simplate detected as %s instead of %s",
			s.Type, SIMPLATE_TYPE_NEGOTIATED)
	}
}

func TestAssignsNoGoPagesToStaticSimplates(t *testing.T) {
	s := SimplateFromString("basic-static.txt", BASIC_STATIC_TXT_SIMPLATE)
	if s.InitPage != nil {
		t.Errorf("Static simplate had init page assigned!: %v", s.InitPage)
	}

	if len(s.LogicPages) > 0 {
		t.Errorf("Static simplate had logic pages assigned!: %v", s.LogicPages)
	}
}

func TestAssignsAnInitPageToRenderedSimplates(t *testing.T) {
	s := SimplateFromString("basic-rendered.txt", BASIC_RENDERED_TXT_SIMPLATE)
	if s.InitPage == nil {
		t.Errorf("Rendered simplate had no init page assigned!: %v", s.InitPage)
	}
}

func TestAssignsOneLogicPageToRenderedSimplates(t *testing.T) {
	s := SimplateFromString("basic-rendered.txt", BASIC_RENDERED_TXT_SIMPLATE)
	if len(s.LogicPages) != 1 {
		t.Errorf("Rendered simplate unexpected number "+
			"of logic pages assigned!: %v", len(s.LogicPages))
	}
}
