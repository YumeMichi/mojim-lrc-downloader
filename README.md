## 魔镜歌词网歌词检索工具

用于下载墨镜歌词网的歌词

## 使用
```
python lrc.py singer [input_song_file] [output_lrc_file]
```

input_song_file 和 output_lrc_file 默认为当前目录下的 songs.txt 和 lrc.txt，可不指定。

## TODO

因为魔镜歌词网不支持分页，并且搜索结果只显示前 200 条，且还不支持歌手歌名联合查询，故需要解决以下几个问题：

1. 针对检索结果前 200 条不包含的歌名，通过查询歌手的相关歌曲记录来查询
2. 歌名、歌手文件读入，歌词检索结果文件读出
3. 其他待补充
