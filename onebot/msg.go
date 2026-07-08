package main

import (
	"encoding/json"
	"time"
)

const (
	maxDownloadMediaChunkBytes = 16 * 1024 * 1024
	maxDownloadMediaBytes      = 64 * 1024 * 1024
)

func Download(rawMsg []byte) error {
	downloadReq := new(DownloadRequest)
	err := json.Unmarshal(rawMsg, downloadReq)
	if err != nil {
		Error("JSON解析失败", "err", err)
		return err
	}

	Info("下载文件", "file_id", downloadReq.FileID, "media_len", len(downloadReq.Media), "cdn_url", downloadReq.CDNURL)
	if downloadReq.CDNURL == "" {
		Warn("跳过缺少 cdn_url 的下载数据", "file_id", downloadReq.FileID, "media_len", len(downloadReq.Media))
		return nil
	}
	if len(downloadReq.Media) == 0 {
		Warn("跳过空下载数据", "file_id", downloadReq.FileID, "cdn_url", downloadReq.CDNURL)
		return nil
	}
	if len(downloadReq.Media) > maxDownloadMediaChunkBytes {
		Warn("跳过过大的下载分片", "file_id", downloadReq.FileID, "cdn_url", downloadReq.CDNURL, "media_len", len(downloadReq.Media), "limit", maxDownloadMediaChunkBytes)
		return nil
	}

	if downloadReqInter, ok := userID2FileMsgMap.Load(downloadReq.CDNURL); ok {
		beforeDownloadReq := downloadReqInter.(*DownloadRequest)
		if beforeDownloadReq.FilePath != "" {
			return nil
		}
		if len(beforeDownloadReq.Media)+len(downloadReq.Media) > maxDownloadMediaBytes {
			userID2FileMsgMap.Delete(downloadReq.CDNURL)
			Warn("清理过大的下载缓存", "file_id", downloadReq.FileID, "cdn_url", downloadReq.CDNURL, "cached_len", len(beforeDownloadReq.Media), "chunk_len", len(downloadReq.Media), "limit", maxDownloadMediaBytes)
			return nil
		}
		if time.Now().UnixMilli()-beforeDownloadReq.LastAppendTime > 10000000 {
			beforeDownloadReq.Media = downloadReq.Media
		} else {
			beforeDownloadReq.Media = append(beforeDownloadReq.Media, downloadReq.Media...)
		}

		beforeDownloadReq.LastAppendTime = time.Now().UnixMilli()
	} else {
		downloadReq.LastAppendTime = time.Now().UnixMilli()
		userID2FileMsgMap.Store(downloadReq.CDNURL, downloadReq)
	}

	return nil
}
