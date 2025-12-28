# lib文件夹说明

运行项目需要在本目录下存放如下文件
- ffmpeg可执行文件：具体为exe/bin由系统类型裁定
- ffprobe可执行文件：具体为exe/bin由系统类型裁定
- whisper相关文件：具体为*.dll/exe或其他由系统类型裁定

作用：
- 视频帧截取
- 视频元数据获取
- 音频转文字

# 下载地址
- ffmpeg（windows）：https://www.gyan.dev/ffmpeg/builds/
- ffmpeg（Linux-Ubuntu）：https://launchpad.net/ubuntu/+source/ffmpeg
- Whisper模型: https://huggingface.co/ggerganov/whisper.cpp/tree/main
- Whisper可执行文件（windows）：https://github.com/ggml-org/whisper.cpp/releases
- Whisper仓库：https://github.com/ggml-org/whisper.cpp
> Linux安装ffmpeg可以直接使用包管理及逆行
> Linux的Whisper需要考虑手动编译源代码