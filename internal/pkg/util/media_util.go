package util

import (
	"Cornerstone/internal/api/config"
	"bufio"
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/liuzl/gocc"
)

// GetSafeContentType 嗅探文件的真实 MIME 类型并校验合法性
func GetSafeContentType(file io.ReadSeeker) (string, error) {
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("读取文件流失败: %w", err)
	}

	contentType := http.DetectContentType(buffer[:n])

	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("重置文件指针失败: %w", err)
	}

	return contentType, nil
}

// GetDimensions 统一使用 ffprobe 获取媒体（图片/视频/动图）的宽高
func GetDimensions(ctx context.Context, mediaUrl string) (int, int, error) {
	ffprobePath := getLibPath(config.Cfg.LibPath.FFprobe)
	cmd := exec.CommandContext(ctx, ffprobePath,
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height",
		"-of", "default=noprint_wrappers=1:nokey=1",
		"-i", mediaUrl,
	)

	out, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("ffprobe 解析媒体属性失败: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return 0, 0, fmt.Errorf("ffprobe 输出数据不足")
	}

	width, _ := strconv.Atoi(strings.TrimSpace(lines[0]))
	height, _ := strconv.Atoi(strings.TrimSpace(lines[1]))

	return width, height, nil
}

// GetDuration 获取视频时长
func GetDuration(ctx context.Context, mediaUrl string) (float64, error) {
	ffprobePath := getLibPath(config.Cfg.LibPath.FFprobe)

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
	ffmpegPath := getLibPath(config.Cfg.LibPath.FFmpeg)
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
	ffmpegPath := getLibPath(config.Cfg.LibPath.FFmpeg)
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

func ResizeImage(imgData io.Reader, width, height, quality int) (io.Reader, error) {
	src, _, err := image.Decode(imgData)
	if err != nil {
		return nil, err
	}

	dst := imaging.Resize(src, width, height, imaging.Lanczos)

	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, dst, &jpeg.Options{Quality: quality})
	return buf, err
}

// AudioStreamToText 音频转文字
func AudioStreamToText(ctx context.Context, mediaUrl string) (string, error) {
	duration, err := GetDuration(ctx, mediaUrl)
	if err != nil {
		return "", err
	}

	// 采样策略
	var segments [][2]float64
	if duration <= 30 {
		segments = append(segments, [2]float64{0, duration})
	} else {
		// 大于30秒，头10s，中10s，尾10s
		segments = append(segments, [2]float64{0, 10})
		segments = append(segments, [2]float64{duration/2 - 5, 10})
		segments = append(segments, [2]float64{duration - 10, 10})
	}

	var fullText strings.Builder
	for _, seg := range segments {
		text, err := transcribeSegment(ctx, mediaUrl, seg[0], seg[1])
		if err != nil {
			return "", err
		}
		if text != "" {
			fullText.WriteString(text + " ")
		}
	}

	res := strings.TrimSpace(fullText.String())
	t2s, err := gocc.New("t2s")
	if err != nil {
		return res, nil
	}
	out, _ := t2s.Convert(res)
	return out, nil
}

// transcribeSegment 转写音频片段
func transcribeSegment(ctx context.Context, url string, start float64, length float64) (string, error) {
	ffmpegPath := getLibPath(config.Cfg.LibPath.FFmpeg)
	whisperPath := getLibPath(config.Cfg.LibPath.Whisper)
	modelPath := getLibPath(config.Cfg.LibPath.WhisperModel)

	ffmpegCmd := exec.CommandContext(ctx, ffmpegPath,
		"-ss", fmt.Sprintf("%.2f", start),
		"-t", fmt.Sprintf("%.2f", length),
		"-i", url,
		"-ar", "16000",
		"-ac", "1",
		"-c:a", "pcm_s16le",
		"-f", "wav",
		"pipe:1",
	)

	whisperCmd := exec.CommandContext(ctx, whisperPath,
		"-m", modelPath,
		"-l", "zh",
		"-f", "-",
		"-nt",
		"--no-prints",
		"--output-file", "--output-*",
	)

	pr, pw := io.Pipe()
	ffmpegCmd.Stdout = pw
	whisperCmd.Stdin = pr
	var outBuf bytes.Buffer
	whisperCmd.Stdout = &outBuf

	if err := ffmpegCmd.Start(); err != nil {
		return "", err
	}
	if err := whisperCmd.Start(); err != nil {
		return "", err
	}

	go func() {
		_ = ffmpegCmd.Wait()
		_ = pw.Close()
	}()

	if err := whisperCmd.Wait(); err != nil {
		return "", err
	}
	return strings.TrimSpace(outBuf.String()), nil
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

func getLibPath(path string) string {
	root := os.Getenv("PROJECT_ROOT")
	if root == "" {
		root, _ = os.Getwd()
		root = filepath.Dir(filepath.Dir(root))
	}
	return filepath.Join(root, path)
}
