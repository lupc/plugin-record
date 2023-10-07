package record

import (
	"io"
	"path/filepath"
	"strconv"
	"time"

	"go.uber.org/zap"
	. "m7s.live/engine/v4"
)

type IRecorder interface {
	ISubscriber
	GetRecorder() *Recorder
	Start(streamPath string) error
	io.Closer
	CreateFile() (FileWr, error)
}

type Recorder struct {
	Subscriber     `json:"-" yaml:"-"`
	SkipTS         uint32
	Record         `json:"-" yaml:"-"`
	File           FileWr `json:"-" yaml:"-"`
	FileName       string // 自定义文件名，分段录像无效
	append         bool   // 是否追加模式
	LastDir        string //记录最后录像目录路径
	lastDirChanged func(dir string)
	// IsRecoding     bool  //正在录像
	// RetryCount     int32 //重试次数
	StartTime  time.Time //开始录像时间
	StreamPath string
	SubType    byte
	RID        string
}

// 最后录像目录路径
func (r *Recorder) GetLastDir() string {
	return r.LastDir
}
func (r *Recorder) SetLastDir(value string) {
	if r.LastDir != value {
		r.LastDir = value
		if r.lastDirChanged != nil {
			r.lastDirChanged(r.LastDir)
		}
	}
}

func (r *Recorder) GetRecorder() *Recorder {
	return r
}

func (r *Recorder) CreateFile() (FileWr, error) {
	return r.createFile()
}

func (r *Recorder) Close() error {
	if r.File != nil {
		return r.File.Close()
	}
	return nil
}

func (r *Recorder) createFile() (f FileWr, err error) {
	filePath := r.getFileName(r.Stream.Path) + r.Ext
	f, err = r.CreateFileFn(filePath, r.append)
	if err == nil {
		r.Info("create file", zap.String("path", filePath))
	} else {
		r.Error("create file", zap.String("path", filePath), zap.Error(err))
	}
	return
}

// 获取记录文件路径
func (r *Recorder) getFileName(streamPath string) (filename string) {
	filename = streamPath
	if r.Fragment == 0 {
		if r.FileName != "" {
			filename = filepath.Join(filename, r.FileName)
		}
	} else {
		filename = filepath.Join(filename, strconv.FormatInt(time.Now().Unix(), 10))
	}
	return
}

// 按年月日生成目录（yyyy-MM/dd）
func (r *Recorder) getLastDir(streamPath string) string {
	var dir = streamPath
	var now = time.Now()
	dir = filepath.Join(dir, now.Format("2006-01/02"))
	r.SetLastDir(dir)
	return r.LastDir
}

// // 获取记录文件路径，
// func (r *Recorder) getFileName2(streamPath string) (filename string) {
// 	filename = streamPath
// 	var now = time.Now()
// 	filename = r.getLastDir(filename)
// 	if r.Fragment == 0 {
// 		if r.FileName != "" {
// 			filename = filepath.Join(filename, r.FileName)
// 		}
// 	} else {
// 		filename = filepath.Join(filename, strconv.FormatInt(now.Unix(), 10))
// 	}
// 	return
// }

func (r *Recorder) start(re IRecorder, streamPath string, subType byte) (err error) {

	if actual, loaded := RecordPluginConfig.recordings.Load(r.ID); loaded {
		if ir, ok := actual.(IRecorder); ok {
			var rcd = ir.GetRecorder()
			if rcd == nil || rcd.StreamPath == "" || rcd.StartTime.IsZero() {
				rcd.stopRecord()
			}
		}
	}

	if _, loaded := RecordPluginConfig.recordings.LoadOrStore(r.ID, re); loaded {
		return ErrRecordExist
	}
	err = plugin.Subscribe(streamPath, re)
	if err == nil {
		r.RID = r.ID
		r.StreamPath = streamPath
		r.SubType = subType
		r.recording[streamPath] = re
		r.Closer = re
		r.Sugar().Debugf("%v开始录制。。", r.ID)
		go func() {
			r.StartTime = time.Now()
			// r.IsRecoding = true
			r.PlayBlock(subType)
			r.stopRecord()
		}()
	}

	return
}

func (r *Recorder) stopRecord() {
	RecordPluginConfig.recordings.Delete(r.ID)
	delete(r.recording, r.StreamPath)
	if r.Closer != nil {
		r.Closer.Close()
	}
	// r.Close()
	// r.IsRecoding = false
	// RecordPluginConfig.retryRecorders.LoadOrStore(r.ID, re)
	r.Sugar().Debugf("%v已停止录制。", r.ID)
}

func (r *Recorder) cut(absTime uint32) {
	if ts := absTime - r.SkipTS; time.Duration(ts)*time.Millisecond >= r.Fragment {
		// r.Debug("切片", zap.Any("ID", r.ID))
		r.SkipTS = absTime
		r.Close()
		if file, err := r.Spesific.(IRecorder).CreateFile(); err == nil {
			r.File = file
			r.Spesific.OnEvent(file)
		} else {
			r.Stop(zap.Any("resion", "切片出错"), zap.Error(err))
		}
		// } else {
		// 	r.Debug("切片条件不符", zap.Any("ts", ts), zap.Any("r.Fragment", r.Fragment))
	}
}

func (r *Recorder) OnEvent(event any) {
	switch v := event.(type) {
	case IRecorder:
		if file, err := r.Spesific.(IRecorder).CreateFile(); err == nil {
			r.File = file
			r.Spesific.OnEvent(file)
		} else {
			r.Stop(zap.Error(err))
		}
	case AudioFrame:
		// 纯音频流的情况下需要切割文件
		if r.Fragment > 0 && r.VideoReader == nil {
			r.cut(v.AbsTime)
		}
	case VideoFrame:
		if r.Fragment > 0 && v.IFrame {
			r.cut(v.AbsTime)
		}
	default:
		r.Subscriber.OnEvent(event)
	}
}
