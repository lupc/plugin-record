package record

import (
	_ "embed"
	"errors"
	"io"
	"sync"

	"go.uber.org/zap"
	. "m7s.live/engine/v4"
	"m7s.live/engine/v4/codec"
	"m7s.live/engine/v4/config"
	"m7s.live/engine/v4/util"
)

type RecordConfig struct {
	DefaultYaml
	config.Subscribe
	Flv        Record
	Mp4        Record
	Fmp4       Record
	Hls        Record
	Raw        Record
	RawAudio   Record
	recordings sync.Map
	FFmpeg     string //ffmpeg路径
}

//go:embed default.yaml
var defaultYaml DefaultYaml
var ErrRecordExist = errors.New("recorder exist")
var RecordPluginConfig = &RecordConfig{
	DefaultYaml: defaultYaml,
	Flv: Record{
		Path:          "record/flv",
		Ext:           ".flv",
		GetDurationFn: getFLVDuration,
	},
	Fmp4: Record{
		Path: "record/fmp4",
		Ext:  ".mp4",
	},
	Mp4: Record{
		Path: "record/mp4",
		Ext:  ".mp4",
	},
	Hls: Record{
		Path: "record/hls",
		Ext:  ".m3u8",
	},
	Raw: Record{
		Path: "record/raw",
		Ext:  ".", // 默认h264扩展名为.h264,h265扩展名为.h265
	},
	RawAudio: Record{
		Path: "record/raw",
		Ext:  ".", // 默认aac扩展名为.aac,pcma扩展名为.pcma,pcmu扩展名为.pcmu
	},
}

var plugin = InstallPlugin(RecordPluginConfig)

func (conf *RecordConfig) OnEvent(event any) {
	switch v := event.(type) {
	case FirstConfig, config.Config:
		conf.Flv.Init()
		conf.Mp4.Init()
		conf.Fmp4.Init()
		conf.Hls.Init()
		conf.Raw.Init()
		conf.RawAudio.Init()

		//启动清理任务
		conf.Hls.StartAutoClean()
		conf.Flv.StartAutoClean()
		conf.Fmp4.StartAutoClean()
		conf.Mp4.StartAutoClean()
		conf.Raw.StartAutoClean()
		conf.RawAudio.StartAutoClean()

		// //启动自动重试
		// conf.Hls.StartRetryRecord()
		// conf.Flv.StartRetryRecord()
		// conf.Fmp4.StartRetryRecord()
		// conf.Mp4.StartRetryRecord()
		// conf.Raw.StartRetryRecord()
		// conf.RawAudio.StartRetryRecord()
	case SEpublish:
		streamPath := v.Target.Path
		if conf.Flv.NeedRecord(streamPath) {
			go NewFLVRecorder().Start(streamPath)
		}
		if conf.Mp4.NeedRecord(streamPath) {
			go NewMP4Recorder().Start(streamPath)
		}
		if conf.Fmp4.NeedRecord(streamPath) {
			go NewFMP4Recorder().Start(streamPath)
		}
		if conf.Hls.NeedRecord(streamPath) {

			plugin.Logger.Debug("SEpublish start record", zap.String("streamPath", streamPath))
			go GetHLSRecorder(streamPath).Start(streamPath)
			// go NewHLSRecorder().Start(streamPath)
		}
		if conf.Raw.NeedRecord(streamPath) {
			go NewRawRecorder().Start(streamPath)
		}
		if conf.RawAudio.NeedRecord(streamPath) {
			go NewRawAudioRecorder().Start(streamPath)
		}
	}
}
func (conf *RecordConfig) getRecorderConfigByType(t string) (recorder *Record) {
	switch t {
	case "flv":
		recorder = &conf.Flv
	case "mp4":
		recorder = &conf.Mp4
	case "fmp4":
		recorder = &conf.Fmp4
	case "hls":
		recorder = &conf.Hls
	case "raw":
		recorder = &conf.Raw
	case "raw_audio":
		recorder = &conf.RawAudio
	}
	return
}

func getFLVDuration(file io.ReadSeeker) uint32 {
	_, err := file.Seek(-4, io.SeekEnd)
	if err == nil {
		var tagSize uint32
		if tagSize, err = util.ReadByteToUint32(file, true); err == nil {
			_, err = file.Seek(-int64(tagSize)-4, io.SeekEnd)
			if err == nil {
				_, timestamp, _, err := codec.ReadFLVTag(file)
				if err == nil {
					return timestamp
				}
			}
		}
	}
	return 0
}

// // 子线程定时检查意外断开的录像重新录制
// func (r *Record) StartRetryRecord() {
// 	if r.Retry == 0 {
// 		return
// 	}
// 	log.Debugf("%v录像自动重试开启。。", r.Path, zap.Any("retry", r.Retry), zap.Any("retryInterval", r.RetryInterval))
// 	go func() {
// 		var interval = r.RetryInterval
// 		if interval <= 0 {
// 			interval = 1 * time.Second
// 		}
// 		for {
// 			var removes = []*Recorder{}
// 			RecordPluginConfig.recordings.Range(func(key, value any) bool {
// 				ir, ok := value.(IRecorder)
// 				if ok {
// 					var recorder = ir.GetRecorder()
// 					if recorder.IsRecoding {
// 						recorder.RetryCount = 0
// 						// removes = append(removes, recorder)
// 					} else {
// 						//重试
// 						if r.Retry < 0 || recorder.RetryCount < r.Retry {
// 							recorder.RetryCount++
// 							log.Debug("录像重试。。", zap.Any("ID", recorder.ID), zap.Any("retryCount", recorder.RetryCount))
// 							recorder.start(recorder.Spesific.(IRecorder), recorder.StreamPath, recorder.SubType)
// 						} else {
// 							log.Debug("超重试次数,停止录像", zap.Any("ID", recorder.ID), zap.Any("retryCount", recorder.RetryCount))
// 							removes = append(removes, recorder)
// 							recorder.Stop(zap.String("resion", "重试超最大次数"))
// 							recorder.Close()
// 						}
// 					}
// 				}

// 				return true
// 			})
// 			if len(removes) > 0 {
// 				for _, rem := range removes {
// 					RecordPluginConfig.recordings.Delete(rem.ID)
// 					delete(r.recording, rem.StreamPath)
// 				}
// 			}
// 			time.Sleep(interval)
// 		}
// 	}()
// }
