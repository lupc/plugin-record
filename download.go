package record

import (
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"m7s.live/engine/v4/log"
	"m7s.live/engine/v4/util"
)

// 下载历史录像
func (p *RecordConfig) API_download(w http.ResponseWriter, r *http.Request) {

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

	var downloadName = fmt.Sprintf("%v-%v-%v.mp4", strings.ReplaceAll(streamPath, "/", "-"), m3u8Info.StartTime.Unix(), m3u8Info.EndTime.Unix())
	w.Header().Set("Content-Type", "video/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment;filename=%v", downloadName))
	var errOut util.Buffer
	cmd := exec.Command(p.FFmpeg, "-i", m3u8Info.Path, "-vcodec", "copy", "-acodec", "copy", "-f", "mpeg", "pipe:1")
	cmd.Stderr = &errOut
	cmd.Stdout = w
	cmd.Run()
	if errOut.CanRead() {
		log.Debugf("ffmpeg:%v", string(errOut))
	}
}
