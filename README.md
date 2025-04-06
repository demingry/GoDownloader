# GoDownloader

This is a tiny file Downloader written in Golang

![](https://github.com/demingry/GoDownloader/blob/main/Screenshot2025-04-06.png)

## Features

- Resume download -> [rfc2616](https://datatracker.ietf.org/doc/html/rfc2616)
- Multi thread -> goroutine.
- BitTorrent download supported.
- TODO: 1.Custom goroutine number 2.io readwriter or channel in code 3.fix BUG 4.Enhance the UI experiences.

### TIPS
1、并发支持不严谨，存在许多问题，需要完善。
2、错误处理没有做，只是ErrorContext一个测试。
3、文件结构乱，后续再整理。
4、种子文件下载没有写完，只是测试。