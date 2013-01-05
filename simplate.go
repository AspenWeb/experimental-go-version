package goaspen

import (
	"fmt"
	"io"
	"mime"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

const (
	SimplateTypeRendered   = "rendered"
	SimplateTypeStatic     = "static"
	SimplateTypeNegotiated = "negotiated"
	SimplateTypeJson       = "json"
)

var (
	SimplateTypes = []string{
		SimplateTypeJson,
		SimplateTypeNegotiated,
		SimplateTypeRendered,
		SimplateTypeStatic,
	}
	simplateTypeTemplates = map[string]*template.Template{
		SimplateTypeJson:       escapedSimplateTemplate(simplateTypeJSONTmpl, "goaspen-gen-json"),
		SimplateTypeRendered:   escapedSimplateTemplate(simplateTypeRenderedTmpl, "goaspen-gen-rendered"),
		SimplateTypeNegotiated: escapedSimplateTemplate(simplateTypeNegotiatedTmpl, "goaspen-gen-negotiated"),
		SimplateTypeStatic:     nil,
	}
	defaultRenderer = "#!go/text/template"
	nonAlNumDash    = regexp.MustCompile("[^-a-zA-Z0-9]")
	vPathPart       = regexp.MustCompile("%([a-zA-Z_][-a-zA-Z0-9_]*)")
)

type simplate struct {
	GenPackage    string
	SiteRoot      string
	Filename      string
	AbsFilename   string
	Type          string
	ContentType   string
	InitPage      *simplatePage
	LogicPage     *simplatePage
	TemplatePages []*simplatePage
}

type simplatePage struct {
	Parent *simplate
	Body   string
	Spec   *simplatePageSpec
}

type simplatePageSpec struct {
	ContentType string
	Renderer    string
}

func newSimplateFromString(packageName,
	siteRoot, filename, content string) (*simplate, error) {

	var err error
	ext := path.Ext(filename)
	hasExt := len(ext) > 0

	absFilename, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}

	filename, err = filepath.Rel(siteRoot, absFilename)
	if err != nil {
		return nil, err
	}

	rawPages := strings.Split(content, "")
	nbreaks := len(rawPages) - 1

	s := &simplate{
		GenPackage:  packageName,
		SiteRoot:    siteRoot,
		Filename:    filename,
		AbsFilename: absFilename,
		Type:        SimplateTypeStatic,
		ContentType: mime.TypeByExtension(ext),
	}

	if nbreaks == 1 || nbreaks == 2 {
		if !hasExt {
			return nil, fmt.Errorf("1 or 2 ^L found in simplate %q!  ",
				"Rendered simplates must have a file extension!", filename)
		}

		s.InitPage, err = newSimplatePage(s, rawPages[0], false)
		if err != nil {
			return nil, err
		}

		s.LogicPage, err = newSimplatePage(s, rawPages[1], false)
		if err != nil {
			return nil, err
		}

		if s.ContentType == "application/json" {
			s.Type = SimplateTypeJson
		} else {
			s.Type = SimplateTypeRendered
			templatePage, err := newSimplatePage(s, rawPages[2], true)
			if err != nil {
				return nil, err
			}

			s.TemplatePages = append(s.TemplatePages, templatePage)
		}

		return s, nil
	}

	if nbreaks > 2 {
		if hasExt {
			return nil, fmt.Errorf("More than 2 ^L found in simplate %q! "+
				"Negotiated simplates must not have a file extension!", filename)
		}

		s.Type = SimplateTypeNegotiated
		s.InitPage, err = newSimplatePage(s, rawPages[0], false)
		if err != nil {
			return nil, err
		}

		s.LogicPage, err = newSimplatePage(s, rawPages[1], false)
		if err != nil {
			return nil, err
		}

		for _, rawPage := range rawPages[2:] {
			templatePage, err := newSimplatePage(s, rawPage, true)
			if err != nil {
				return nil, err
			}

			s.TemplatePages = append(s.TemplatePages, templatePage)
		}

		return s, nil
	}

	return s, nil
}

func (me *simplate) FirstTemplatePage() *simplatePage {
	if len(me.TemplatePages) > 0 {
		return me.TemplatePages[0]
	}

	return nil
}

func (me *simplate) Execute(wr io.Writer) (err error) {
	defer func(err *error) {
		r := recover()
		if r != nil {
			*err = fmt.Errorf("%v", r)
		}
	}(&err)

	debugf("Executing to %+v\n", wr)
	*(&err) = simplateTypeTemplates[me.Type].Execute(wr, me)
	return
}

func (me *simplate) escapedFilename() string {
	fn := filepath.Clean(me.Filename)
	lessDots := strings.Replace(fn, ".", "-DOT-", -1)
	lessSlashes := strings.Replace(lessDots, "/", "-SLASH-", -1)
	lessSpaces := strings.Replace(lessSlashes, " ", "-SPACE-", -1)
	lessPercents := strings.Replace(lessSpaces, "%", "-PCT-", -1)
	squeaky := nonAlNumDash.ReplaceAllString(lessPercents, "-")
	return strings.Replace(squeaky, "--", "-", -1)
}

func (me *simplate) OutputName() string {
	if me.Type == SimplateTypeStatic {
		return me.Filename
	}

	return me.escapedFilename() + ".go"
}

func (me *simplate) FuncName() string {
	escaped := me.escapedFilename()
	parts := strings.Split(escaped, "-")
	for i, part := range parts {
		var capitalized []string
		capitalized = append(capitalized, strings.ToUpper(string(part[0])))
		capitalized = append(capitalized, strings.ToLower(part[1:]))
		parts[i] = strings.Join(capitalized, "")
	}

	return strings.Join(parts, "")
}

func (me *simplate) ConstName() string {
	escaped := me.escapedFilename()
	uppered := strings.ToUpper(escaped)
	return strings.Replace(uppered, "-", "_", -1)
}

func newSimplatePageSpec(simplate *simplate, specline string) (*simplatePageSpec, error) {
	sps := &simplatePageSpec{
		ContentType: simplate.ContentType,
		Renderer:    defaultRenderer,
	}

	switch simplate.Type {
	case SimplateTypeStatic:
		return &simplatePageSpec{}, nil
	case SimplateTypeJson:
		return sps, nil
	case SimplateTypeRendered:
		renderer := specline
		if len(renderer) < 1 {
			renderer = defaultRenderer
		}

		sps.Renderer = renderer
		return sps, nil
	case SimplateTypeNegotiated:
		parts := strings.Fields(specline)
		nParts := len(parts)

		if nParts < 1 || nParts > 2 {
			return nil, fmt.Errorf("A negotiated resource specline "+
				"must have one or two parts: #!renderer media/type. Yours is %q",
				specline)
		}

		if nParts == 1 {
			sps.ContentType = parts[0]
			sps.Renderer = defaultRenderer
			return sps, nil
		} else {
			sps.ContentType = parts[0]
			sps.Renderer = parts[1]
			return sps, nil
		}
	}

	return nil, fmt.Errorf("Can't make a page spec "+
		"for simplate type %q", simplate.Type)
}

func newSimplatePage(simplate *simplate, rawPage string, needsSpec bool) (*simplatePage, error) {
	spec := &simplatePageSpec{}
	var err error

	specline := ""
	body := rawPage

	if needsSpec {
		parts := strings.SplitN(rawPage, "\n", 2)
		specline = parts[0]
		body = parts[1]

		spec, err = newSimplatePageSpec(simplate,
			strings.TrimSpace(strings.Replace(specline, "", "", -1)))
		if err != nil {
			return nil, err
		}
	}

	sp := &simplatePage{
		Parent: simplate,
		Body:   body,
		Spec:   spec,
	}
	return sp, nil
}
