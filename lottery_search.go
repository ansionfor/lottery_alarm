package main

import (
        "github.com/gogf/gf/encoding/gjson"
        "github.com/gogf/gf/frame/g"
        "github.com/gogf/gf/os/gtime"
        "github.com/gogf/gf/os/gcron"
        "github.com/go-gomail/gomail"
        "crypto/tls"
        "strings"
        "fmt"
        "time"
)

const (
        getUrl            = "http://apis.juhe.cn/lottery/query"
        getKey            = "ca132f7c54c50fd347147b561f38daa2"
        lotteryId         = "ssq"
        succCode  float64 = 0
        crontabTime       = "0 20 21 * * TUE,THU,SUN"              // 开奖时间，周二，周四，周日晚21点15分
        checkIntvalMin    = 10                                       // 间隔n分钟检查一次结果
        checkMaxTimes     = 5                                        // 最多检查n次，n次都失败则结束
        smtpHost          = "smtp.qq.com"
        smtpPort          = 587
        smtpEmail         = "xx@qq.com"
        smtpSecCode       = ""                       // qq邮箱授权码
        toEmail           = "xx@qq.com"
)

var (
        // 已选号码
        destNums     = g.Config().GetArray("lottery.num")
        // 赢奖规则
        winningRules = [][][]int{{{6, 1}}, {{6, 0}}, {{5, 1}}, {{5, 0}, {4, 1}}, {{4, 0}, {3, 1}}, {{2, 1}, {1, 1}, {0, 1}}}
)

func main() {
        gcron.Add(crontabTime, func() {
                json := getResult()
                date := gtime.Now().Format("Y-m-d")
                i := checkMaxTimes
                fmt.Println(json)
                for !checkResult(json, date) && i > 0 {
                        i--
                        time.Sleep(time.Duration(checkIntvalMin)*time.Minute)
                }
        })
        select {}
}

func getResult() *gjson.Json {
        res := g.Client().GetContent(getUrl, g.Map{"lottery_id": lotteryId, "key": getKey})
        json := gjson.New(res)
        return json
}

func checkResult(j *gjson.Json, date string) bool {
        reason := j.Get("error_code")
        if reason != succCode {
                g.Log().Info("查询失败", reason, succCode)
                return false
        }
        resDate := j.Get("result.lottery_date")
        if resDate != date {
                g.Log().Info("查询时间未到", resDate, date)
                return false
        }
        resNum := j.Get("result.lottery_res")
        numSlice := strings.Split(resNum.(string), ",")
        redBall := numSlice[:6]
        blueBall := numSlice[6]

        destString := ""
        resString := strings.Join(numSlice, " ")
        msg := ""
        for _, val := range destNums {
                vTmp := val.(string)
                v := strings.Split(vTmp, ",")
                destString += strings.Join(v, " ") + "<br>"
                redBallDest := v[:6]
                blueBallDest := v[6]
                avaiableRed := intersect(redBall, redBallDest)
                blueCount := 0
                if (blueBall == blueBallDest) {
                        blueCount = 1
                }

                // 判断是否中奖
                redCount := len(avaiableRed)
                if redCount == 0 && blueCount == 0 {
                        continue
                }
                isWin := false
                for level, val := range winningRules {
                        if isWin {
                                break
                        }
                        for _, value := range val {
                                if value[0] <= redCount && value[1] == blueCount {
                                        msg = fmt.Sprintf("恭喜你，中了 %d 等奖! ", level)
                                        sendMsg("双色球中奖啦!", msg)
                                        isWin = true
                                        break
                                }
                        }
                }
        }
        msg = fmt.Sprintf("今晚双色球<br>开奖号码：<br>%s<br>已选号码：<br>%s", resString, destString)
        sendMsg("今晚双色球开奖结果", msg)
        return true
}


// 中将了，发通知
func sendMsg(title, msg string) {
        d := gomail.NewDialer(smtpHost, smtpPort, smtpEmail, smtpSecCode)
        d.TLSConfig = &tls.Config{InsecureSkipVerify: true}

        m := gomail.NewMessage()
        m.SetHeader("From", toEmail)
        m.SetHeader("To", toEmail)
        m.SetHeader("Subject", title)
        m.SetBody("text/html", msg)
        if err := d.DialAndSend(m); err != nil {
                g.Log().Info("DialAndSend err %v:", err)
                panic(err)
        }
}

//求交集
func intersect(slice1, slice2 []string) []string {
        m := make(map[string]int)
        nn := make([]string, 0)
        for _, v := range slice1 {
                m[v]++
        }

        for _, v := range slice2 {
                times, _ := m[v]
                if times == 1 {
                        nn = append(nn, v)
                }
        }
        return nn
}
