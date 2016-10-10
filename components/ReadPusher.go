package components

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chengchenginc/go-logfile-read-push/config"
	redis "github.com/garyburd/redigo/redis"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const (
	defaultBufSize = 4096
)

type Reader interface {
	Open() error
	ReadLines() error
	Handle(line []byte) error //出错或者读取原有到时间行停止
	Flush() error             //刷新游标数据
}

type PushInfo struct {
	Key     string `json:"key"`
	Time    string `json:"time"`
	Message string `json:"message"`
}

type Pusher interface {
	Push(pi PushInfo) error
}

type Closer interface {
	Close() error
}

type ReadPusher interface {
	Reader
	Pusher
	Closer
}

type ReadRedisPusher struct {
	filepath                  string
	redisCon                  redis.Conn
	database                  string
	metricsLastTurnExcuteTime string
	metricsLastExcuteTime     string
}

func NewReadPusher(rconfig config.RedisConfig, file config.LogFileConfig) (rrp *ReadRedisPusher, err error) {
	conn, err := redis.Dial("tcp", rconfig.Host+":"+strconv.Itoa(rconfig.Port))
	if err != nil {
		fmt.Println("redis can't connection!")
		return nil, err
	}
	rrp = &ReadRedisPusher{
		filepath: file.FilePath,
		redisCon: conn,
		database: rconfig.Database,
	}
	return rrp, nil
}

func (rrp *ReadRedisPusher) Open() error {
	return nil
}

func (rrp *ReadRedisPusher) Push(pi PushInfo) error {
	b, err := json.Marshal(pi)
	if err != nil {
		return err
	}
	_, err = rrp.redisCon.Do("LPUSH", rrp.database, b)
	if err != nil {
		return err
	}
	return nil
}

func (rrp *ReadRedisPusher) Close() error {
	return nil
}

func (rrp *ReadRedisPusher) Handle(line []byte) error {
	strline := string(line)
	r, err := regexp.Compile(`\[[^[]+\]`)
	if err != nil {
		fmt.Printf("There is a problem with regexp.\n")
		return err
	}
	matches := r.FindAllString(strline, -1)
	if len(matches) == 4 && matches[1] == "[:error]" { //0:时间 1:error 2:pid 3:client
		//第一行读取
		if len(rrp.metricsLastExcuteTime) == 0 {
			rrp.metricsLastExcuteTime = matches[0]
		}

		//第一次会从某位读取文件开头
		if matches[0] == rrp.metricsLastTurnExcuteTime {
			fmt.Println("find last excute time ,stop!")
			return errors.New("find last excute time ,stop!")
		}
		error_msg := r.ReplaceAllString(strline, "")
		time := strings.TrimLeft(matches[0], "[")
		time = strings.TrimRight(time, "]")
		pi := PushInfo{
			Key:     time,
			Time:    time,
			Message: error_msg,
		}
		_ = rrp.Push(pi)
	}
	return nil
}

func (rrp *ReadRedisPusher) ReadLines() (err error) {
	f, e := os.Stat(rrp.filepath)
	if e == nil {
		size := f.Size()
		var fi *os.File
		fi, err = os.Open(rrp.filepath)
		if err == nil {
			b := make([]byte, defaultBufSize)
			sz := int64(defaultBufSize)
			bTail := bytes.NewBuffer([]byte{})
			istart := size
			isFileTop := false
			for {
				if istart < defaultBufSize {
					sz = istart
					istart = 0
					isFileTop = true
				} else {
					istart -= sz
				}
				_, err := fi.Seek(istart, os.SEEK_SET)
				if err != nil {
					break
				}
				mm, e := fi.Read(b)
				if e == nil && mm > 0 {
					j := mm
					//line
					for i := mm - 1; i >= 0; i-- {
						if b[i] == '\n' {
							bLine := bytes.NewBuffer([]byte{})
							bLine.Write(b[i+1 : j])
							j = i
							if bTail.Len() > 0 {
								bLine.Write(bTail.Bytes())
								bTail.Reset()
							}
							if bLine.Len() > 0 { //skip last "\n"
								err = rrp.Handle(bLine.Bytes())
								if err != nil {
									return err
								}
							}
						}
					}
					if j > 0 {
						if istart == 0 {
							bLine := bytes.NewBuffer([]byte{})
							bLine.Write(b[:j])
							if bTail.Len() > 0 {
								bLine.Write(bTail.Bytes())
								bTail.Reset()
							}
						} else {
							bb := make([]byte, bTail.Len())
							copy(bb, bTail.Bytes())
							bTail.Reset()
							bTail.Write(b[:j])
							bTail.Write(bb)
						}
					}

					if isFileTop == true {
						//重置循环
						rrp.metricsLastTurnExcuteTime = rrp.metricsLastExcuteTime
						rrp.metricsLastExcuteTime = ""
						fmt.Println("reach to file top" + rrp.metricsLastTurnExcuteTime)
						return errors.New("reach to file top!")
					}
				}
			}
		}
		defer fi.Close()
	}
	return
}
