package record

import (
	"os"
	"time"

	"m7s.live/engine/v4/log"
)

// 自动清理录像
func (r *Recorder) StartAutoClean() {

	// 每日2点执行清理任务
	now := time.Now()
	var hour = 2
	dateTime := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location())

	go DailyCron(dateTime, func() {
		r.execClean()
	})
	log.Infof("自动清理任务已启动... 每天%v点执行,自动清理%v天前的文件。", hour, r.AutoClean)
}

func (r *Recorder) execClean() {

	//统一处理错误
	defer func() {
		if err := recover(); err != nil {
			switch err := err.(type) {
			case string:
				log.Errorf("清理历史录像文件出错！%v", err)
			case error:
				log.Errorf("清理历史录像文件出错！%v", err)
			}
		}
	}()

	if r.AutoClean == 0 {
		return
	}
	log.Infof("自动清理任务执行...")
	//递归扫描所有文件
	var cleanPath = r.Path
	var err = CleanFiles(cleanPath, r.AutoClean)
	if err != nil {
		panic(err)
	}
}

// 递归清理文件和文件夹
func CleanFiles(pathname string, days int32) error {

	fis, err := os.ReadDir(pathname)
	if err != nil {
		log.Errorf("读取文件目录出错！pathname=%v, err=%v \n", pathname, err)
		return err
	}

	if len(fis) == 0 {
		//删除空目录
		err = os.Remove(pathname)
		if err != nil {
			log.Infof("目录已删除：%v", pathname)
		} else {
			log.Errorf("目录删除出错：%v,%v", pathname, err)
		}

		return nil
	}

	// 所有文件/文件夹
	for _, fi := range fis {
		fullname := pathname + "/" + fi.Name()
		// 是文件夹则递归进入获取;是文件，则压入数组
		if fi.IsDir() {
			err := CleanFiles(fullname, days)
			if err != nil {
				log.Errorf("清理目录出错！fullname=%v, err=%v", fullname, err)
				return err
			}
		} else {
			var finfo, err = os.Stat(fullname)
			if err != nil {
				log.Errorf("获取文件信息出错：%v,%v", fullname, err)
			} else if needClean(finfo, days) {
				err = os.Remove(fullname)
				if err != nil {
					log.Infof("文件已删除：%v", fullname)
				} else {
					log.Errorf("文件删除出错：%v,%v", fullname, err)
				}
			}
		}
	}

	return nil
}

func needClean(finfo os.FileInfo, days int32) bool {
	var y, m, d = time.Now().Date()
	var date = time.Date(y, m, d, 0, 0, 0, 0, time.Now().Location())
	var isDel = (date.Sub(finfo.ModTime()).Hours() > float64(days*24))
	return isDel
}

// DailyCron 每日指定时间 执行任意个无参任务
// 若今天已经超过了执行时间则等到第二天的指定时间再执行任务
func DailyCron(dateTime time.Time, tasks ...func()) {
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), dateTime.Hour(), dateTime.Minute(), dateTime.Second(), dateTime.Nanosecond(), dateTime.Location())
		// 检查是否超过当日的时间
		if next.Sub(now) < 0 {
			next = now.Add(time.Hour * 24)
			next = time.Date(next.Year(), next.Month(), next.Day(), dateTime.Hour(), dateTime.Minute(), dateTime.Second(), dateTime.Nanosecond(), dateTime.Location())
		}
		// 阻塞到执行时间
		t := time.NewTimer(next.Sub(now))
		<-t.C
		// 执行的任务内容
		for _, task := range tasks {
			task()
		}
	}
}
