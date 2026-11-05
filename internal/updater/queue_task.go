package updater

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// QueueUpdateOptions selects the components checked by the queue's resource
// update task.
type QueueUpdateOptions struct {
	Core     bool
	Base     bool
	Overlay  bool
	Download bool
}

// QueueUpdateResult reports current/latest versions and the staged component.
type QueueUpdateResult struct {
	CoreCurrent    string
	CoreLatest     string
	BaseCurrent    string
	BaseLatest     string
	OverlayCurrent string
	OverlayLatest  string
	Staged         string
	Message        string
}

// CheckForQueueTask checks the selected updates and optionally stages the
// first available one. Only one update can be staged at a time.
func (s *Service) CheckForQueueTask(ctx context.Context, options QueueUpdateOptions) (QueueUpdateResult, error) {
	versions := s.Versions()
	result := QueueUpdateResult{
		CoreCurrent:    versions.Core,
		BaseCurrent:    versions.Base,
		OverlayCurrent: versions.Overlay,
	}
	var checked int
	var problems []string

	coreAvailable := false
	baseAvailable := false
	overlayAvailable := false
	var coreRelease, baseRelease, overlayRelease ReleaseInfo

	// Metadata checks must not stall the queue when a proxy or the direct
	// fallback route stops responding.
	checkContext := func() (context.Context, context.CancelFunc) {
		return context.WithTimeout(ctx, 25*time.Second)
	}

	if options.Core {
		checkCtx, cancel := checkContext()
		release, err := s.Check(checkCtx)
		cancel()
		if err != nil {
			problems = append(problems, "MaaCore 检查失败："+err.Error())
		} else {
			checked++
			coreRelease = release
			result.CoreLatest = release.Tag
			coreAvailable = result.CoreCurrent == "" || !strings.EqualFold(result.CoreCurrent, release.Tag)
		}
	}
	if options.Base {
		checkCtx, cancel := checkContext()
		release, err := s.CheckBase(checkCtx)
		cancel()
		if err != nil {
			problems = append(problems, "底包检查失败："+err.Error())
		} else {
			checked++
			baseRelease = release
			result.BaseLatest = release.Tag
			baseAvailable = result.BaseCurrent == "" || !strings.EqualFold(result.BaseCurrent, release.Tag)
		}
	}
	if options.Overlay {
		checkCtx, cancel := checkContext()
		release, err := s.CheckOverlay(checkCtx)
		cancel()
		if err != nil {
			problems = append(problems, "覆盖检查失败："+err.Error())
		} else {
			checked++
			overlayRelease = release
			result.OverlayLatest = release.Tag
			overlayAvailable = result.OverlayCurrent == "" || !strings.EqualFold(result.OverlayCurrent, release.Tag)
		}
	}
	if checked == 0 && len(problems) > 0 {
		return result, fmt.Errorf("%s", strings.Join(problems, "；"))
	}
	if len(problems) > 0 {
		result.Message = strings.Join(problems, "；")
	}

	switch {
	case coreAvailable:
		if options.Download {
			if err := s.DownloadCore(ctx, coreRelease.Tag); err != nil {
				return result, err
			}
			result.Staged = "core"
		}
		result.Message = "MaaCore 有新版本 " + coreRelease.Tag
	case baseAvailable:
		if options.Download {
			if err := s.DownloadBase(ctx); err != nil {
				return result, err
			}
			result.Staged = "resource-base"
		}
		result.Message = "资源底包有新版本 " + baseRelease.Tag
	case overlayAvailable:
		if options.Download {
			if err := s.DownloadOverlay(ctx); err != nil {
				return result, err
			}
			result.Staged = "resource-overlay"
		}
		result.Message = "资源覆盖有新版本 " + overlayRelease.Tag
	default:
		if result.Message == "" {
			result.Message = "检查完成，所有组件已是最新"
		}
	}
	return result, nil
}
