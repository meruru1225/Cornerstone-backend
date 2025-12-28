package util

import (
	"Cornerstone/internal/api/config"
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"

	"github.com/liuzl/gocc"
)

// GetDuration 获取视频时长
func GetDuration(ctx context.Context, mediaUrl string) (float64, error) {
	ffprobePath := config.Cfg.LibPath.FFprobe

	cmd := exec.CommandContext(ctx, ffprobePath,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		"-i", mediaUrl,
	)

	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe 解析失败: %w", err)
	}

	return strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
}

// GetAudioStream 获取视频的音频流
func GetAudioStream(ctx context.Context, mediaUrl string) (io.ReadCloser, error) {
	ffmpegPath := config.Cfg.LibPath.FFmpeg
	cmd := exec.CommandContext(ctx, ffmpegPath,
		"-i", mediaUrl,
		"-vn",
		"-ar", "16000",
		"-ac", "1",
		"-c:a", "pcm_s16le",
		"-f", "wav",
		"pipe:1",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err = cmd.Start(); err != nil {
		return nil, err
	}

	return stdout, nil
}

// GetImageFrames 获取视频帧
func GetImageFrames(ctx context.Context, mediaUrl string, duration float64) ([]io.Reader, error) {
	ffmpegPath := config.Cfg.LibPath.FFmpeg
	fps := 5.0 / duration

	cmd := exec.CommandContext(ctx, ffmpegPath,
		"-i", mediaUrl,
		"-vf", fmt.Sprintf("fps=%f", fps),
		"-f", "image2pipe", "-vcodec", "mjpeg", "pipe:1",
	)

	stdout, _ := cmd.StdoutPipe()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	var frames []io.Reader
	scanner := bufio.NewScanner(stdout)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)
	scanner.Split(splitJPEG)

	for scanner.Scan() {
		data := scanner.Bytes()
		tmp := make([]byte, len(data))
		copy(tmp, data)
		frames = append(frames, bytes.NewReader(tmp))
	}

	if err := cmd.Wait(); err != nil {
		fmt.Printf("FFmpeg Error: %s\n", stderr.String())
		return nil, err
	}
	return frames, nil
}

// AudioStreamToText 将音频流转换为文本
func AudioStreamToText(ctx context.Context, mediaUrl string) (string, error) {
	ffmpegPath := config.Cfg.LibPath.FFmpeg
	whisperPath := config.Cfg.LibPath.Whisper
	modelPath := config.Cfg.LibPath.WhisperModel

	// FFmpeg 从 URL 获取音频并输出标准 16kHz WAV 管道流
	ffmpegCmd := exec.CommandContext(ctx, ffmpegPath,
		"-i", mediaUrl,
		"-ar", "16000", "-ac", "1", "-c:a", "pcm_s16le", "-f", "wav", "pipe:1")

	// Whisper执行
	whisperCmd := exec.CommandContext(ctx, whisperPath,
		"-m", modelPath,
		"-l", "zh",
		"-f", "-",
		"-nt",
		"--no-prints",
		"--output-file", "--output-*",
	)

	// 建立管道连接
	pr, pw := io.Pipe()
	ffmpegCmd.Stdout = pw
	whisperCmd.Stdin = pr

	var outBuf bytes.Buffer
	whisperCmd.Stdout = &outBuf

	// 启动进程
	if err := ffmpegCmd.Start(); err != nil {
		return "", err
	}
	if err := whisperCmd.Start(); err != nil {
		return "", err
	}

	// 异步监控生产者
	go func() {
		_ = ffmpegCmd.Wait()
		_ = pw.Close()
	}()

	// 等待 Whisper 识别完成
	if err := whisperCmd.Wait(); err != nil {
		return "", err
	}

	// 返回结果，同时尽可能返回简体
	res := strings.TrimSpace(outBuf.String())
	t2s, err := gocc.New("t2s")
	if err != nil {
		return res, nil
	}
	out, err := t2s.Convert(res)
	if err != nil {
		return res, nil
	}
	return out, nil
}

// splitJPEG 辅助函数：基于特征码切割 JPEG 流
func splitJPEG(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.Index(data, []byte{0xFF, 0xD9}); i >= 0 {
		return i + 2, data[0 : i+2], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}
