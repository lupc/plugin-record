package record

import (
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type FileWr interface {
	io.Reader
	io.Writer
	io.Seeker
	io.Closer
}
type VideoFileInfo struct {
	Path     string
	Size     int64
	Duration uint32
}

type Record struct {
	Ext           string //文件扩展名
	Path          string //存储文件的目录
	AutoRecord    bool
	Filter        string
	Fragment      time.Duration //分片大小，0表示不分片
	AutoClean     int32         //自动清理N天前的录像，0表示不清理，30表示30天前
	Retry         int32         //意外停止自动重试次数，-1:无限重试，0:不重试，
	RetryInterval time.Duration //重试时间间隔,最小1秒
	filterReg     *regexp.Regexp
	fs            http.Handler
	CreateFileFn  func(filename string, append bool) (FileWr, error) `json:"-" yaml:"-"`
	GetDurationFn func(file io.ReadSeeker) uint32                    `json:"-" yaml:"-"`
	// recording     map[string]IRecorder
}

func (r *Record) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.fs.ServeHTTP(w, req)
}

func (r *Record) NeedRecord(streamPath string) bool {
	return r.AutoRecord && (r.filterReg == nil || r.filterReg.MatchString(streamPath))
}

func (r *Record) Init() {
	// r.recording = make(map[string]IRecorder)
	os.MkdirAll(r.Path, 0777)
	if r.Filter != "" {
		r.filterReg = regexp.MustCompile(r.Filter)
	}
	r.fs = http.FileServer(http.Dir(r.Path))
	r.CreateFileFn = func(filename string, append bool) (file FileWr, err error) {
		filePath := filepath.Join(r.Path, filename)
		if err = os.MkdirAll(filepath.Dir(filePath), 0777); err != nil {
			return file, err
		}
		file, err = os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0777)
		return
	}

}

func (r *Record) Tree(dstPath string, level int) (files []*VideoFileInfo, err error) {
	var dstF *os.File
	dstF, err = os.Open(dstPath)
	if err != nil {
		return
	}
	defer dstF.Close()
	fileInfo, err := dstF.Stat()
	if err != nil {
		return
	}
	if !fileInfo.IsDir() { //如果dstF是文件
		if r.Ext == "." || path.Ext(fileInfo.Name()) == r.Ext {
			p := strings.TrimPrefix(dstPath, r.Path)
			p = strings.ReplaceAll(p, "\\", "/")
			var duration uint32
			if r.GetDurationFn != nil {
				duration = r.GetDurationFn(dstF)
			}
			files = append(files, &VideoFileInfo{
				Path:     strings.TrimPrefix(p, "/"),
				Size:     fileInfo.Size(),
				Duration: duration,
			})
		}
		return
	} else { //如果dstF是文件夹
		var dir []os.FileInfo
		dir, err = dstF.Readdir(0) //获取文件夹下各个文件或文件夹的fileInfo
		if err != nil {
			return
		}
		for _, fileInfo = range dir {
			var _files []*VideoFileInfo
			_files, err = r.Tree(filepath.Join(dstPath, fileInfo.Name()), level+1)
			if err != nil {
				return
			}
			files = append(files, _files...)
		}
		return
	}

}
