package pegel

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const TimeLayout string = "2006-01-02 15:04"

func download(url string) ([]byte, error) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.0.0 Safari/537.36")
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	statusOK := response.StatusCode >= 200 && response.StatusCode < 300
	if !statusOK {
		return nil, fmt.Errorf("non-OK HTTP status: %d", response.StatusCode)
	}

	body, _ := io.ReadAll(response.Body)

	return body, nil
}

var rePegel = regexp.MustCompile(`^\s*\['00389','Ebnet','Dreisam',3,'(\d+)','cm','(\d\d)\.(\d\d)\.(\d\d\d\d) (\d\d):(\d\d) [A-Z]+',`)

func parseLine(line string) (TimeValue, error) {
	m := rePegel.FindStringSubmatch(line)
	if m == nil {
		return TimeValue{}, fmt.Errorf("cannot parse data line: %v", line)
	}

	pegel, err := strconv.Atoi(m[1])
	if err != nil {
		return TimeValue{}, fmt.Errorf("cannot parse data line (invalid pegel '%s'): %v", m[1], line)
	}

	year, err := strconv.Atoi(m[4])
	if err != nil {
		return TimeValue{}, fmt.Errorf("cannot parse data line (invalid year '%s'): %v", m[4], line)
	}
	month, err := strconv.Atoi(m[3])
	if err != nil {
		return TimeValue{}, fmt.Errorf("cannot parse data line (invalid month '%s'): %v", m[3], line)
	}
	day, err := strconv.Atoi(m[2])
	if err != nil {
		return TimeValue{}, fmt.Errorf("cannot parse data line (invalid month '%s'): %v", m[2], line)
	}

	hh, err := strconv.Atoi(m[5])
	if err != nil {
		return TimeValue{}, fmt.Errorf("cannot parse data line (invalid hour '%s'): %v", m[5], line)
	}
	mm, err := strconv.Atoi(m[6])
	if err != nil {
		return TimeValue{}, fmt.Errorf("cannot parse data line (invalid minute '%s'): %v", m[6], line)
	}

	return TimeValue{time.Date(year, time.Month(month), day, hh, mm, 0, 0, time.UTC), int64(pegel)}, nil
}

func getMtime(filePath string) (time.Time, error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return time.Time{}, err
	}

	return stat.ModTime(), nil
}

func downloadOrCache(url, cacheFilePath string) ([]byte, error) {
	maxAge := 5 * time.Minute
	if mtime, err := getMtime(cacheFilePath); err == nil && mtime.After(time.Now().Add(-maxAge)) {
		return os.ReadFile(cacheFilePath)
	}

	data, err := download(url)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(cacheFilePath), 0770); err != nil {
		return nil, fmt.Errorf("while creating cache folder: %w", err)
	}

	err = os.WriteFile(cacheFilePath, data, 0644)
	if err != nil {
		return nil, fmt.Errorf("while creating cache file: %w", err)
	}

	return data, nil
}

type TimeValue struct {
	TimeStamp time.Time
	Value     int64
}

type PegelData struct {
	Pegel TimeValue
	Trend []int64
}

func GetPegel(dataDir string) (TimeValue, error) {
	cacheFilePath := filepath.Join(dataDir, "cache")
	data, err := downloadOrCache("https://www.hvz.baden-wuerttemberg.de/js/hvz_peg_stmn.js", cacheFilePath)
	if err != nil {
		return TimeValue{}, fmt.Errorf("cannot download pegel data: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, " ['00389',") {
			return parseLine(line)
		}
	}

	return TimeValue{}, fmt.Errorf("cannot find ebnet/dreisam in pegel data")
}

var reLine = regexp.MustCompile(`^(\d\d\d\d-\d\d-\d\d \d\d:\d\d);(\d+)$`)

func readData(path string) ([]TimeValue, error) {
	data := make([]TimeValue, 0)

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return data, nil
	}

	f, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return data, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if len(line) == 0 {
			continue
		}
		m := reLine.FindStringSubmatch(line)
		if m == nil {
			return data, fmt.Errorf("cannot parse line: %v", line)
		}

		t, err := time.Parse(TimeLayout, m[1])
		if err != nil {
			return data, fmt.Errorf("cannot parse line (date): %v", line)
		}

		v, err := strconv.Atoi(m[2])
		if err != nil {
			return data, fmt.Errorf("cannot parse line (value): %v", line)
		}

		data = append(data, TimeValue{t, int64(v)})
	}

	if err := sc.Err(); err != nil {
		return data, err
	}

	return data, nil
}

func writeData(path string, data []TimeValue) error {
	if err := os.MkdirAll(filepath.Dir(path), 0770); err != nil {
		return fmt.Errorf("while creating data dir: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, d := range data {
		s := fmt.Sprintf("%s;%d\n", d.TimeStamp.Format(TimeLayout), d.Value)
		if _, err := f.WriteString(s); err != nil {
			return err
		}
	}

	return nil
}

func GetPegelData(dataDir string) (PegelData, error) {
	pegel, err := GetPegel(dataDir)
	if err != nil {
		return PegelData{}, err
	}

	historyPath := filepath.Join(dataDir, "history")
	data, err := readData(historyPath)
	if err != nil {
		return PegelData{}, err
	}

	ldata := len(data)
	if ldata == 0 || pegel.TimeStamp.After(data[ldata-1].TimeStamp) {
		data = append(data, pegel)
	}

	if err := writeData(historyPath, data); err != nil {
		return PegelData{}, err
	}

	ldata = len(data)
	trend := make([]int64, 0, 5)
	for i := ldata - 1; i >= 1 && len(trend) < 5; i = i - 1 {
		trend = append(trend, data[i].Value-data[i-1].Value)
	}

	return PegelData{pegel, trend}, nil
}
