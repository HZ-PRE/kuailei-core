package hcore

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/HZ-PRE/kuailei-core/v2/config"
	"golang.org/x/net/proxy"

	"github.com/sagernet/sing-box/option"
)

func getRandomAvailblePort() uint16 {
	// TODO: implement it
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}
	defer listener.Close()
	return uint16(listener.Addr().(*net.TCPAddr).Port)
}

func RunInstanceString(ctx context.Context, sdmSettings *config.SdmOptions, proxiesInput string) (*SdmInstance, error) {
	if sdmSettings == nil {
		sdmSettings = config.DefaultSdmOptions()
	}

	singconfigs, err := config.ParseConfig(ctx, &config.ReadOptions{Content: proxiesInput}, true, sdmSettings, false)
	if err != nil {
		return nil, err
	}
	return RunInstance(ctx, sdmSettings, singconfigs)
}

func RunInstance(ctx context.Context, sdmSettings *config.SdmOptions, singconfig *option.Options) (*SdmInstance, error) {
	if sdmSettings == nil {
		sdmSettings = config.DefaultSdmOptions()
	}
	sdmSettings.EnableClashApi = false
	sdmSettings.InboundOptions.MixedPort = getRandomAvailblePort()
	sdmSettings.InboundOptions.EnableTun = false
	sdmSettings.InboundOptions.EnableTunService = false
	sdmSettings.InboundOptions.SetSystemProxy = false
	sdmSettings.InboundOptions.TProxyPort = 0
	sdmSettings.InboundOptions.DirectPort = 0
	sdmSettings.InboundOptions.RedirectPort = 0
	sdmSettings.Region = "other"
	sdmSettings.BlockAds = false
	sdmSettings.LogFile = "/dev/null"

	finalConfigs, err := config.BuildConfig(ctx, sdmSettings, &config.ReadOptions{Options: singconfig})
	if err != nil {
		return nil, err
	}

	instance, err := NewService(ctx, *finalConfigs)
	if err != nil {
		return nil, err
	}

	<-time.After(250 * time.Millisecond)
	hservice := &SdmInstance{
		StartedService: instance,
		ListenPort:     sdmSettings.InboundOptions.MixedPort}
	hservice.PingCloudflare()
	return hservice, nil
}

// dialer, err := s.libbox.GetInstance().Router().Dialer(context.Background())

func (s *SdmInstance) Close() error {
	return s.StartedService.CloseService()
}

func (s *SdmInstance) GetContent(url string) (string, error) {
	return s.ContentFromURL("GET", url, 10*time.Second)
}

func (s *SdmInstance) ContentFromURL(method string, url string, timeout time.Duration) (string, error) {
	if method == "" {
		return "", fmt.Errorf("empty method")
	}
	if url == "" {
		return "", fmt.Errorf("empty url")
	}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return "", err
	}

	dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("127.0.0.1:%d", s.ListenPort), nil, proxy.Direct)
	if err != nil {
		return "", err
	}

	transport := &http.Transport{
		Dial: dialer.Dial,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return "", fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if body == nil {
		return "", fmt.Errorf("empty body")
	}

	return string(body), nil
}

func (s *SdmInstance) PingCloudflare() (time.Duration, error) {
	return s.Ping("http://cp.cloudflare.com")
}

// func (s *SdmService) RawConnection(ctx context.Context, url string) (net.Conn, error) {
// 	return
// }

func (s *SdmInstance) PingAverage(url string, count int) (time.Duration, error) {
	if count <= 0 {
		return -1, fmt.Errorf("count must be greater than 0")
	}

	var sum int
	real_count := 0
	for i := 0; i < count; i++ {
		delay, err := s.Ping(url)
		if err == nil {
			real_count++
			sum += int(delay.Milliseconds())
		} else if real_count == 0 && i > count/2 {
			return -1, fmt.Errorf("ping average failed")
		}

	}
	return time.Duration(sum / real_count * int(time.Millisecond)), nil
}

func (s *SdmInstance) Ping(url string) (time.Duration, error) {
	startTime := time.Now()
	_, err := s.ContentFromURL("HEAD", url, 4*time.Second)
	if err != nil {
		return -1, err
	}
	duration := time.Since(startTime)
	return duration, nil
}
