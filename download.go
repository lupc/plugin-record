package record

import (
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"m7s.live/engine/v4/log"
	"m7s.live/engine/v4/util"
)

// 设置跨域
func setupCORS(w *http.ResponseWriter) {
	// (*w).Header().Set("Access-Control-Allow-Origin", "*")
	// // (*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	// (*w).Header().Set("Access-Control-Allow-Headers", "*")
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type,authorization,Authorization")
	(*w).Header().Add("Access-Control-Allow-Credentials", "true")
	(*w).Header().Add("Access-Control-Max-Age", "86400000")
	(*w).Header().Add("Access-Control-Allow-Methods", "OPTIONS, HEAD, POST, GET, PUT, DELETE")

}

// 下载历史录像
func (p *RecordConfig) API_download(w http.ResponseWriter, r *http.Request) {

	setupCORS(&w)

	//统一处理错误
	defer func() {
		if err := recover(); err != nil {
			returnErrRes(&w, err, 404)
		}
	}()
	log.Infof("下载录像请求: %v,", r.URL)

	if p.FFmpeg == "" {
		panic("ffmpeg没有配置!")
	}

	var q = r.URL.Query()
	var startTime = q.Get("st")
	var endTime = q.Get("et")
	var streamPath = q.Get("path")

	var m3u8Info = p.genVod(startTime, endTime, streamPath)
	if m3u8Info == nil {
		panic("生成HLS点播文件失败！")
	}

	var downloadName = fmt.Sprintf("%v-%v-%v.ts", strings.ReplaceAll(streamPath, "/", "-"), m3u8Info.StartTime.Unix(), m3u8Info.EndTime.Unix())
	// w.Header().Set("Content-Type", "video/octet-stream")
	w.Header().Set("Content-Type", "video/mpeg")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment;filename=%v", downloadName))
	var errOut util.Buffer
	cmd := exec.Command(p.FFmpeg, "-i", m3u8Info.Path, "-vcodec", "copy", "-acodec", "copy", "-f", "mpegts", "pipe:1")
	cmd.Stderr = &errOut
	cmd.Stdout = w
	cmd.Run()
	if errOut.CanRead() {
		log.Debugf("ffmpeg:%v", string(errOut))
	}
}
