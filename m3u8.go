package record

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

// ts文件信息
type TsInfo struct {
	EXTINF   string
	FileName string
	Time     time.Time //时间
	Len      float64   //时长 秒
}

// m3u8文件信息
type M3u8FileInfo struct {
	Head      string
	StartTime time.Time //开始时间
	EndTime   time.Time //结束时间
	TsFiles   []*TsInfo //ts文件信息
	JoinPath  string    //ts文件所在目录，生成m3u8文件内容时会在ts文件名前拼接该路径
	Path      string    //m3u8文件路径
}

const (
	M3U_HEAD = `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-MEDIA-SEQUENCE:0`
)

// 根据m3u8文件路径新建m3u8文件信息
func New(fileName string) (*M3u8FileInfo, error) {
	var m3u8 = M3u8FileInfo{}
	var errOut error
	data, err_read := os.ReadFile(fileName) // 读取文件
	if err_read != nil {
		return &m3u8, err_read
	}
	var fileContent = string(data)
	if len(fileContent) > 0 {
		var lines = strings.Split(strings.ReplaceAll(fileContent, "\r\n", "\n"), "\n")
		var isOverHead = false
		var totalLen = 0.0
		for i, line := range lines {
			if strings.HasPrefix(line, "#EXTINF") {
				isOverHead = true
				var ts = &TsInfo{EXTINF: line, FileName: strings.ReplaceAll(lines[i+1], "\\", "/")}
				//解析时间
				var tsTimeStr = strings.ReplaceAll(path.Base(ts.FileName), ".ts", "") //获取时间戳
				tsTime, errParse := strconv.ParseInt(tsTimeStr, 10, 64)
				if errParse == nil {
					ts.Time = time.Unix(tsTime, 0)
				}
				//解析时长
				var lenStr = strings.ReplaceAll(ts.EXTINF, "#EXTINF:", "")
				lenStr = strings.TrimRight(lenStr, ",")
				lenStr = strings.Trim(lenStr, " ")
				len, errParse := strconv.ParseFloat(lenStr, 64)
				if errParse == nil {
					ts.Len = len
					totalLen += ts.Len
				}
				m3u8.TsFiles = append(m3u8.TsFiles, ts)
			} else if !isOverHead {
				m3u8.Head += line + "\n"
			}
		}

		if len(m3u8.TsFiles) > 0 {
			// //修正时长
			// totalLen = 0
			// for i, ts := range m3u8.TsFiles {
			// 	if i < (len(m3u8.TsFiles) - 1) {
			// 		var tsNext = m3u8.TsFiles[i+1]
			// 		ts.Len = tsNext.Time.Sub(ts.Time).Abs().Seconds()
			// 	} else {
			// 		ts.Len = m3u8.TsFiles[i-1].Len
			// 	}
			// 	totalLen += ts.Len
			// }

			m3u8.StartTime = m3u8.TsFiles[0].Time
			m3u8.EndTime = m3u8.StartTime.Add(time.Second * time.Duration(totalLen))
		}
	}
	return &m3u8, errOut
}

func Make(tsInfos []*TsInfo) (info *M3u8FileInfo, err error) {

	var tsLen = len(tsInfos)
	if tsInfos == nil || tsLen == 0 {
		err = fmt.Errorf("TS列表为空！")
		return
	}

	var st = tsInfos[0].Time
	var last = tsInfos[tsLen-1]
	var et = last.Time.Add(time.Duration(last.Len))
	var head = M3U_HEAD + fmt.Sprintf("\n#EXT-X-TARGETDURATION:%v\n", math.Ceil(tsInfos[0].Len))
	info = &M3u8FileInfo{
		StartTime: st,
		EndTime:   et,
		Head:      head,
		TsFiles:   tsInfos,
	}
	return
}

// 截取其中某个时间段作为一个新的m3u8内容返回
func (m *M3u8FileInfo) Tack(st, et time.Time) (*M3u8FileInfo, error) {
	if st == m.StartTime && et == m.EndTime {
		return m, nil
	}
	if et.Before(st) {
		return nil, errors.New("结束时间必须大于开始时间！")
	}
	if st.Before(m.StartTime) {
		st = m.StartTime
	}
	if et.After(m.EndTime) {
		et = m.EndTime
	}
	var newInfo = M3u8FileInfo{StartTime: st, EndTime: et, Head: m.Head}
	if len(m.TsFiles) > 0 {
		for _, ts := range m.TsFiles {
			var st2 = st.Add(time.Second * -time.Duration(ts.Len))
			var et2 = et.Add(time.Second * time.Duration(ts.Len))
			if st2.Before(ts.Time) && et2.After(ts.Time) {
				newInfo.TsFiles = append(newInfo.TsFiles, ts)
			}
		}

		if len(newInfo.TsFiles) > 0 {
			newInfo.StartTime = newInfo.TsFiles[0].Time
			var lastTs = newInfo.TsFiles[len(newInfo.TsFiles)-1]
			newInfo.EndTime = lastTs.Time.Add(time.Second * time.Duration(lastTs.Len))
		}

	}
	return &newInfo, nil
}

func (m *M3u8FileInfo) ToFileContent() string {
	var sb = strings.Builder{}
	sb.WriteString(m.Head)
	if len(m.TsFiles) > 0 {
		for _, ts := range m.TsFiles {
			sb.WriteString(fmt.Sprintf("#EXTINF:%v,", ts.Len))
			sb.WriteString("\n")
			sb.WriteString(m.JoinPath)
			sb.WriteString(ts.FileName)
			sb.WriteString("\n")
		}
	}
	sb.WriteString("#EXT-X-ENDLIST")
	return sb.String()
}
