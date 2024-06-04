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
	LastCutTime    time.Time //最后切片时间
	IsRecording    bool      //是否正在录制
	Record         `json:"-" yaml:"-"`
	File           FileWr `json:"-" yaml:"-"`
	FileName       string // 自定义文件名，分段录像无效
	append         bool   // 是否追加模式
	LastDir        string `json:"-" yaml:"-"` //记录最后录像目录路径
	lastDirChanged func(dir string)
	// RetryCount     int32 //重试次数
	StartTime time.Time //开始录像时间
	// StopTime   time.Time //停止录像时间
	StreamPath      string `json:"-" yaml:"-"`
	SubType         byte
	RID             string
	BeforeStartFunc func() `json:"-" yaml:"-"` //在开始前执行
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

// 定时检测是否需要结束录像
func (r *Recorder) pollingCheck() {
	for r.IsRecording {
		if r.IsCutNotChange() {
			r.Stop(zap.String("reason", "cut time not change"))
			r.stopRecord()
			r.Logger.Debug("IsCutNotChange true", zap.Any("RID", r.RID), zap.Any("startTime", r.StartTime), zap.Any("cutTime", r.LastCutTime), zap.Any("Fragment", r.Record.Fragment))
		}
		time.Sleep(r.Record.Fragment)
	}
}

func (r *Recorder) IsCutNotChange() bool {
	var checkTime = r.Record.Fragment * 5
	return time.Since(r.StartTime) > checkTime && time.Since(r.LastCutTime) > checkTime
}

func (r *Recorder) start(re IRecorder, streamPath string, subType byte) (err error) {

	defer func() {
		err := recover() //内置函数，可以捕捉到函数异常
		if err != nil {
			//这里是打印错误，还可以进行报警处理，例如微信，邮箱通知
			plugin.Logger.Error("开始录像出错", zap.Any("path", streamPath), zap.Any("err", err))
		}
	}()

	if _, isExist := RecordPluginConfig.recordings.Load(r.ID); isExist {
		return ErrRecordExist
	}

	err = plugin.Subscribe(streamPath, re)
	if err == nil {
		r.IsRecording = true
		RecordPluginConfig.recordings.Store(r.ID, re)
		if r.BeforeStartFunc != nil {
			r.BeforeStartFunc()
		}
		r.RID = r.ID
		r.StreamPath = streamPath
		r.SubType = subType
		// r.recording[streamPath] = re
		r.Closer = re
		r.Sugar().Debugf("%v开始录制。。", r.ID)
		go func() {
			defer func() {
				err := recover() //内置函数，可以捕捉到函数异常
				if err != nil {
					//这里是打印错误，还可以进行报警处理，例如微信，邮箱通知
					plugin.Logger.Error("开始录像出错（协程）", zap.Any("path", streamPath), zap.Any("err", err))
				}
			}()

			r.StartTime = time.Now()
			r.PlayBlock(subType)
			r.Sugar().Debugf("%v阻塞播放结束", r.ID)

			r.stopRecord() //其他地方调stop用会崩溃
		}()

		go r.pollingCheck()
	}

	return
}

func (r *Recorder) stopRecord() {
	if !r.IsRecording {
		return
	}
	r.IsRecording = false
	// r.Stop(zap.String("reason", "record stoped"))
	RecordPluginConfig.recordings.Delete(r.ID)
	// delete(r.recording, r.StreamPath)
	// if r.Closer != nil {
	// 	r.Closer.Close()
	// }

	// r.StopTime = time.Now()
	r.Sugar().Debugf("%v已停止录制。", r.ID)
}

func (r *Recorder) cut(absTime uint32) {
	if ts := absTime - r.SkipTS; time.Duration(ts)*time.Millisecond >= r.Fragment {
		// r.Debug("切片", zap.Any("ID", r.ID))
		r.SkipTS = absTime
		r.LastCutTime = time.Now()
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
