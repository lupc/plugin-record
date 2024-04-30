package record

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	. "m7s.live/engine/v4"
	"m7s.live/engine/v4/util"
)

// WebAPI统一返回格式
type ApiRes struct {
	IsSuc bool   //是否成功
	Code  int    //返回编码
	Msg   string //应答消息
}

func (conf *RecordConfig) API_list(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	t := query.Get("type")
	var files []*VideoFileInfo
	var err error
	recorder := conf.getRecorderConfigByType(t)
	if recorder == nil {
		for _, t = range []string{"flv", "mp4", "fmp4", "hls", "raw", "raw_audio"} {
			recorder = conf.getRecorderConfigByType(t)
			var fs []*VideoFileInfo
			if fs, err = recorder.Tree(recorder.Path, 0); err == nil {
				files = append(files, fs...)
			}
		}
	} else {
		files, err = recorder.Tree(recorder.Path, 0)
	}

	if err == nil {
		var bytes []byte
		if bytes, err = json.Marshal(files); err == nil {
			w.Write(bytes)
		}
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (conf *RecordConfig) API_start(w http.ResponseWriter, r *http.Request) {

	//统一处理错误
	defer func() {
		if err := recover(); err != nil {
			http.Error(w, fmt.Sprint(err), http.StatusInternalServerError)
		}
	}()

	query := r.URL.Query()
	streamPath := query.Get("streamPath")
	fileName := query.Get("fileName")
	fragment := query.Get("fragment")
	if streamPath == "" {
		http.Error(w, "no streamPath", http.StatusBadRequest)
		return
	}
	t := query.Get("type")
	//var id string
	var err error
	var irecorder IRecorder
	switch t {
	case "":
		t = "flv"
		fallthrough
	case "flv":
		irecorder = NewFLVRecorder()
	case "mp4":
		irecorder = NewMP4Recorder()
	case "fmp4":
		irecorder = NewFMP4Recorder()
	case "hls":
		irecorder = GetHLSRecorder(streamPath)
		// irecorder = NewHLSRecorder()
	case "raw":
		irecorder = NewRawRecorder()
	case "raw_audio":
		irecorder = NewRawAudioRecorder()
	default:
		http.Error(w, "type not supported", http.StatusBadRequest)
		return
	}
	recorder := irecorder.GetRecorder()
	if fragment != "" {
		recorder.Fragment, _ = time.ParseDuration(fragment)
	}
	recorder.FileName = fileName
	recorder.append = query.Get("append") != ""
	// plugin.Logger.Debug("visit record/api/start begin", zap.Any("url", r.URL))
	err = irecorder.Start(streamPath)
	// plugin.Logger.Debug("visit record/api/start end", zap.Any("url", r.URL))
	// id = recorder.ID
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//fmt.Fprintf(w, id)
}

func (conf *RecordConfig) API_list_recording(w http.ResponseWriter, r *http.Request) {
	util.ReturnFetchValue(func() (recordings []any) {
		conf.recordings.Range(func(key, value any) bool {
			recordings = append(recordings, value)
			return true
		})
		return
	}, w, r)
}

func (conf *RecordConfig) API_stop(w http.ResponseWriter, r *http.Request) {
	if recorder, ok := conf.recordings.Load(r.URL.Query().Get("id")); ok {
		recorder.(ISubscriber).Stop()
		// var rcd = recorder.(IRecorder).GetRecorder()
		// RecordPluginConfig.recordings.Delete(rcd.ID)
		// delete(rcd.recording, rcd.StreamPath)
		w.Write([]byte("ok"))
		return
	}
	http.Error(w, "no such recorder", http.StatusBadRequest)
}
