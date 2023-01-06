## 魔镜歌词网歌词检索工具

用于下载墨镜歌词网的歌词

## 安装
```
go install github.com/YumeMichi/mojim-lrc-downloader@latest
```

## 使用
1. 配置好 `lrc-in.txt`，格式为 `歌名,歌手,歌词编号`，其中歌词编号仅在存在相同歌手歌名情况下需要指定（根据提示操作）。
2. 运行 `mojim-lrc-downloader` 等待歌词下载完成即可。

首次运行会生成配置文件 `config.yaml`，根据需要进行修改即可。
```
proxy:
    enabled: true
    protocol: socks5
    ip: 127.0.0.1
    port: 1081
lrc:
    load_file_name: lrc-in.txt
    save_file_name: lrc-out.txt
```
