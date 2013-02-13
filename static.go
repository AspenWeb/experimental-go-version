package aspen

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path"
	"strings"
)

type websiteStaticHandler struct {
	w  *Website
	nh pipelineHandler
}

type directoryListing struct {
	RequestPath string
	FullPath    string
	Entries     []*directoryListingEntry
}

type directoryListingEntry struct {
	RequestPath string
	LinkName    string
	FileInfo    os.FileInfo
}

type serveDirError struct {
	Path string
}

func (me *websiteStaticHandler) NextHandler() pipelineHandler {
	return me.nh
}

func (me *websiteStaticHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	debugf("Handling static request for %q", req.URL.Path)

	fullPath := path.Join(me.w.WwwRoot, strings.TrimLeft(req.URL.Path, "/"))
	req.Header.Set(pathTransHeader, fullPath)

	err := me.serveStatic(w, req)
	if err == nil {
		return
	}

	debugf("Serving static failed, so checking if directory listing is appropriate")

	if _, ok := err.(*serveDirError); ok && me.w.ListDirs {
		err = me.serveDirListing(w, req)
		if err == nil {
			return
		}
	}

	if strings.HasSuffix(req.URL.Path, "/favicon.ico") {
		debugf("Serving canned favicon response for %q", req.URL.Path)
		w.Header().Set("Content-Type", "image/x-icon")
		w.WriteHeader(http.StatusOK)
		w.Write(faviconIco)
		return
	}

	debugf("Falling through to 404 for %q!", req.URL.Path)
	serve404(w, req)
}

func (me *websiteStaticHandler) String() string {
	return fmt.Sprintf("*websiteStaticHandler{w.WwwRoot: %q}", me.w.WwwRoot)
}

func (me *websiteStaticHandler) serveStatic(w http.ResponseWriter, req *http.Request) error {
	fullPath, err := me.findStaticPath(req)
	if err != nil {
		return err
	}

	debugf("Found static path %q from root %q and request path %q",
		fullPath, me.w.WwwRoot, req.URL.Path)

	fi, err := os.Stat(fullPath)
	if err != nil {
		return err
	}

	if fi.IsDir() {
		return &serveDirError{Path: fullPath}
	}

	ctype := mime.TypeByExtension(path.Ext(fullPath))
	if strings.HasPrefix(ctype, "text/") && !strings.Contains(ctype, "charset=") {
		ctype = fmt.Sprintf("%v; charset=utf-8", ctype)
	}

	outf, err := os.Open(fullPath)
	if err != nil {
		debugf("Could not open %q", fullPath)
		return err
	}

	defer outf.Close()

	w.Header().Set("Content-Length", fmt.Sprintf("%v", fi.Size()))
	w.Header().Set("Content-Type", ctype)
	w.WriteHeader(http.StatusOK)
	io.Copy(w, outf)

	return nil
}

func (me *websiteStaticHandler) serveDirListing(w http.ResponseWriter,
	req *http.Request) error {

	debugf("Serving directory listing for %q", req.URL.Path)

	fullPath := req.Header.Get(pathTransHeader)
	if len(fullPath) == 0 {
		fullPath = path.Join(me.w.WwwRoot, strings.TrimLeft(req.URL.Path, "/"))
	}

	fi, err := os.Stat(fullPath)
	if err != nil {
		return err
	}

	if !fi.IsDir() {
		return fmt.Errorf("%q is not a directory!", fullPath)
	}

	dirListing, err := newDirListing(req.URL.Path, fullPath)
	if err != nil {
		return err
	}

	htmlListing, err := dirListing.Html()
	if err != nil {
		return err
	}

	w.Header().Set("Content-Length", fmt.Sprintf("%v", len(htmlListing)))
	w.Header().Set("Content-Type",
		fmt.Sprintf("text/html; charset=%v", me.w.CharsetDynamic))
	w.WriteHeader(http.StatusOK)
	w.Write(htmlListing)

	return nil
}

func (me *websiteStaticHandler) findStaticPath(req *http.Request) (string, error) {
	fullPath := req.Header.Get(pathTransHeader)
	if len(fullPath) == 0 {
		fullPath = path.Join(me.w.WwwRoot, strings.TrimLeft(req.URL.Path, "/"))
	}

	fi, err := os.Stat(fullPath)
	if _, ok := err.(*os.PathError); ok {
		debugf("Failed to stat %q: %v", fullPath, err)
		return "", err
	}

	if fi.IsDir() {
		debugf("%q is a directory", fullPath)
		debugf("Looking for candidate index files.  Configured indices = %+v",
			me.w.Indices)

		for _, idx := range me.w.Indices {
			if len(idx) == 0 {
				continue
			}

			tryFullPath := path.Join(fullPath, idx)

			debugf("Checking for candidate index file at %q", tryFullPath)
			fi, err := os.Stat(tryFullPath)
			if err != nil || fi.IsDir() {
				continue
			}

			debugf("Found candidate index file at %q", tryFullPath)
			return tryFullPath, nil
		}
	}

	return fullPath, nil
}

func newDirListing(requestPath, dirPath string) (*directoryListing, error) {
	entries, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	dlEntries := []*directoryListingEntry{}

	for _, ent := range entries {
		reqPath := path.Join(requestPath, ent.Name())
		linkName := ent.Name()

		if ent.IsDir() {
			reqPath = reqPath + "/"
			linkName = linkName + "/"
		}

		dlEnt := &directoryListingEntry{
			RequestPath: reqPath,
			LinkName:    linkName,
			FileInfo:    ent,
		}

		dlEntries = append(dlEntries, dlEnt)
	}

	dl := &directoryListing{
		RequestPath: requestPath,
		FullPath:    dirPath,
		Entries:     dlEntries,
	}
	return dl, nil
}

func (me *directoryListing) Html() ([]byte, error) {
	var buf bytes.Buffer
	err := directoryListingTmpl.Execute(&buf, me)
	if err != nil {
		return []byte(""), err
	}

	return buf.Bytes(), nil
}

func (me *directoryListing) WebParentDir() string {
	if me.RequestPath == "/" {
		return "/"
	}

	parDir := path.Dir(strings.TrimRight(me.RequestPath, "/"))

	if parDir == "/" {
		return "/"
	}

	return parDir + "/"
}

func (me *serveDirError) Error() string {
	return fmt.Sprintf("Directory %q cannot be served!", me.Path)
}
