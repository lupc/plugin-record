package record

import (
	_ "embed"
	"errors"
	"io"
	"sync"

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

//go:embed ui/*
//go:embed README.md
var ui embed.FS

func init() {
	InstallPlugin(&PluginConfig{
		Name:   "Record",
		Type:   PLUGIN_SUBSCRIBER,
		Config: &config,
		Run:    run,
		UIFile: &ui,
		HotConfig: map[string]func(interface{}){
			"AutoPublish": func(v interface{}) {
				config.AutoPublish = v.(bool)
			},
			"AutoRecord": func(v interface{}) {
				config.AutoRecord = v.(bool)
			},
		},
	})
}
func run() {
	OnSubscribeHooks.AddHook(onSubscribe)
	OnPublishHooks.AddHook(onPublish)
	os.MkdirAll(config.Path, 0755)
	http.HandleFunc("/record/flv/list", func(writer http.ResponseWriter, r *http.Request) {
		if files, err := tree(config.Path, 0); err == nil {
			var bytes []byte
			if bytes, err = json.Marshal(files); err == nil {
				writer.Write(bytes)
			} else {
				writer.Write([]byte("{\"err\":\"" + err.Error() + "\"}"))
			}
		} else {
			writer.Write([]byte("{\"err\":\"" + err.Error() + "\"}"))
		}
		if conf.Mp4.NeedRecord(streamPath) {
			go NewMP4Recorder().Start(streamPath)
		}
		if conf.Fmp4.NeedRecord(streamPath) {
			go NewFMP4Recorder().Start(streamPath)
		}
		if conf.Hls.NeedRecord(streamPath) {
			go NewHLSRecorder().Start(streamPath)
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
