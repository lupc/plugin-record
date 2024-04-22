package record

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"m7s.live/engine/v4/log"
	// . "m7s.live/engine/v4"
)

// 点播配置

type VodRes struct {
	ApiRes
	StartTime time.Time //开始时间
	EndTime   time.Time //结束时间
	Url       string    //回放地址
}

// func (p *VodConfig) API_Vod(w http.ResponseWriter, r *http.Request) {
// 	fileName := "record/hls/live/64/test.m3u8"  // txt文件路径
// 	data, err_read := ioutil.ReadFile(fileName) // 读取文件
// 	if err_read != nil {
// 		// fmt.Println("文件读取失败！")
// 		var msg = fmt.Sprintf("文件读取失败！%v\n", err_read)
// 		w.Write([]byte(msg))
// 		return
// 	}
// 	w.Write(data)
// }

func toTime(ts string) time.Time {
	tsInt, errParse := strconv.ParseInt(ts, 10, 64)
	if errParse == nil {
		return time.Unix(tsInt, 0)
	}
	return time.Time{}
}

// 返回错误应答
func returnErrRes(w *http.ResponseWriter, err any, stateCode int) {

	var res = ApiRes{}
	res.IsSuc = false
	if stateCode == 0 {
		stateCode = 200
	}
	res.Code = stateCode
	(*w).WriteHeader(int(stateCode))
	switch err := err.(type) {
	case string:
		res.Msg = err
	case error:
		res.Msg = err.Error()
	}
	log.Error(err)
	var resJson, _ = json.Marshal(res)
	(*w).Write([]byte(resJson))
}

func GetCurrentDirectory() string {
	//返回绝对路径  filepath.Dir(os.Args[0])去除最后一个元素的路径
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}

	//将\替换成/
	return strings.Replace(dir, "\\", "/", -1)
}

// 超找时间段交集最大的m3u8
func findM3u8Info(dir string, st, et time.Time) *M3u8FileInfo {

	// dir = path.Join(GetCurrentDirectory(), dir)
	// log.Infof("record路径：%v", dir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		panic(err)
	}

	var m3u8Info *M3u8FileInfo
	var maxIntersection int64 = 0
	for _, entry := range entries {
		info, err := entry.Info()
		if err == nil && path.Ext(info.Name()) == ".m3u8" {

			relPath := path.Join(dir, info.Name())
			var info, err = NewM3u8Info(relPath)
			if err == nil {

				var curIntersection int64 = 0
				if info.StartTime.Before(st) && info.EndTime.After(et) {
					//包含
					m3u8Info = info
					break
				} else if st.Before(info.StartTime) {

					if et.After(info.StartTime) {
						if et.Before(info.EndTime) {
							//前段交集
							curIntersection = et.Sub(info.StartTime).Microseconds()
						} else {
							//中间交集
							curIntersection = info.EndTime.Sub(info.StartTime).Microseconds()
						}

					}
				} else if et.After(info.EndTime) {

					if st.Before(info.EndTime) {
						if st.After(info.StartTime) {
							//后段交集
							curIntersection = info.EndTime.Sub(st).Microseconds()
						} else {
							//中间交集
							curIntersection = info.EndTime.Sub(info.StartTime).Microseconds()
						}

					}

				}

				if curIntersection > int64(maxIntersection) {
					m3u8Info = info
					maxIntersection = curIntersection
				}
			}
			// tsFiles = append(tsFiles, info)
			// fmt.Printf("info: %v\n", info)
		}
	}
	return m3u8Info
}

// 找出目录下在时间段内的ts文件
func findTsInfos(dir string, st, et time.Time) (tsFiles []*TsInfo) {

	// var date1 = time.Date(st.Year(),st.Month(),st.Day(),0,0,0,0,time.Local)
	// var date2 = time.Date(et.Year(),et.Month(),et.Day(),0,0,0,0,time.Local)
	// var dateDirs []string
	// for y := st.Year(); y <= et.Year(); y++ {
	// 	for m := st.Month(); m <= et.Month(); m++ {
	// 		for d := st.Day(); d <= et.Day(); d++ {
	// 			dateDirs = append(dateDirs, filepath.Join(dir, fmt.Sprintf("%v-%v/%v", y, m, d)))
	// 		}
	// 	}
	// }

	// for _, dateDir := range dateDirs {
	entries, err := os.ReadDir(dir)
	if err == nil {
		for _, entry := range entries {
			info, err := entry.Info()
			if err == nil && path.Ext(info.Name()) == ".m3u8" {
				var timeStr = strings.ReplaceAll(path.Base(info.Name()), ".m3u8", "") //获取时间戳
				if len(timeStr) >= 8 {
					y, err := strconv.Atoi(timeStr[0:4])
					if err != nil {
						continue
					}
					m, err := strconv.Atoi(timeStr[4:6])
					if err != nil {
						continue
					}
					d, err := strconv.Atoi(timeStr[6:8])
					if err != nil {
						continue
					}
					var fileCreateTime = time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.Local)
					//var fileModTime = info.ModTime() //windows不准，获取到缓存的修改时间
					var fileModTime = fileCreateTime.Add(24 * time.Hour)
					log.LocaleLogger.Debug("m3u8信息", zap.Any("dir", dir), zap.Any("file", info.Name()), zap.Any("creatTime", fileCreateTime), zap.Time("modTime", fileModTime))
					if st.Before(fileModTime) && et.After(fileCreateTime) {
						relPath := path.Join(dir, info.Name())
						var info, err = NewM3u8Info(relPath)
						if err == nil {
							//log.LocaleLogger.Debug("m3u8内容", zap.Any("startTime", info.StartTime), zap.Time("endTime", info.EndTime), zap.Int("tsFilesCount", len(info.TsFiles)))
							for _, ts := range info.TsFiles {
								if (st.Before(ts.Time) || st.Equal(ts.Time)) && et.After(ts.Time) {
									tsFiles = append(tsFiles, ts)
								}
							}
						}
					}

				}

			}
		}
	}
	// }

	return
}

// 生成点播
func (p *RecordConfig) genVod(startTime, endTime, streamPath string) *M3u8FileInfo {
	var st = toTime(startTime)
	var et = toTime(endTime)

	log.Infof("尝试生成HLS点播, st=%v,et=%v,path=%v", st, et, streamPath)
	var findDir = path.Join(p.Hls.Path, streamPath)
	// var m3u8Info = findM3u8Info(tsDir, st, et)
	var tsInfos = findTsInfos(findDir, st, et)
	newM3u8Info, err := MakeM3u8Info(tsInfos)
	if err != nil {
		panic(err)
	}

	newM3u8Info.JoinPath = "../"
	var vodDir = path.Join(findDir, "vod")
	if _, err := os.Stat(vodDir); os.IsNotExist(err) {
		// 先创建文件夹
		err = os.Mkdir(vodDir, 0777)
		if err != nil {
			panic(err)
		}
		// 再修改权限
		err = os.Chmod(vodDir, 0777)
		if err != nil {
			panic(err)
		}
	}
	var fileName = fmt.Sprintf("%v-%v.m3u8", newM3u8Info.StartTime.Unix(), newM3u8Info.EndTime.Unix())
	var m3u8Path = path.Join(vodDir, fileName)
	err = os.WriteFile(m3u8Path, []byte(newM3u8Info.ToFileContent()), 0666)
	if err != nil {
		panic(err)
	}
	log.Infof("HLS点播文件已生成: %v,", m3u8Path)
	newM3u8Info.Path = m3u8Path
	// res.Url = fmt.Sprintf("http://%v/%v", r.Host, strings.ReplaceAll(filePath, "/hls", ""))
	// res.StartTime = newInfo.StartTime
	// res.EndTime = newInfo.EndTime

	return newM3u8Info
}

// 生成HLS点播文件API接口
func (p *RecordConfig) API_vod_hls(w http.ResponseWriter, r *http.Request) {

	//统一处理错误
	defer func() {
		if err := recover(); err != nil {
			returnErrRes(&w, err, 400)
		}
	}()

	log.Infof("HLS点播请求：%v", r.URL)
	var res = VodRes{}
	var q = r.URL.Query()
	// fmt.Printf("p: %v\n", p)

	var startTime = q.Get("st")
	var endTime = q.Get("et")
	var streamPath = q.Get("path")

	var m3u8Info = p.genVod(startTime, endTime, streamPath)

	if m3u8Info == nil {
		panic("HLS点播失败！")
	}

	res.Url = fmt.Sprintf("http://%v/%v", r.Host, strings.ReplaceAll(m3u8Info.Path, "/hls", ""))
	res.StartTime = m3u8Info.StartTime
	res.EndTime = m3u8Info.EndTime
	res.IsSuc = true
	res.Msg = "HLS点播成功"

	resJson, err := json.Marshal(res)
	if err != nil {
		panic(err)
	}

	w.Write([]byte(resJson))

}

// var plugin = InstallPlugin(new(VodConfig))
