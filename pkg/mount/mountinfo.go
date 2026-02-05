package mount

import (
	"fmt"
	"gaiasec-nodeagent/pkg/util"
	"strconv"
	"strings"
)

type PartitionStat struct {
	Device     string `json:"device"`
	Mountpoint string `json:"mountpoint"`
	Options    string `json:"options"`
	Fstype     string `json:"fstype"`
	Major      int    `json:"major"`
	Minor      int    `json:"minor"`
}

func GetPartitions(mountinfo string) ([]PartitionStat, error) {
	lines, err := util.ReadLines(mountinfo)
	if err != nil {
		return nil, err
	}
	ret := make([]PartitionStat, len(lines))
	for _, line := range lines {
		var d PartitionStat

		// a line of 1/mountinfo has the following structure:
		// 36 35 98:0 /mnt1 /mnt2 rw,noatime master:1 - ext3 /dev/root rw,errors=continue
		// (maj:min) (fs_type) (mount_opts) (fs_device) (mount_point) (physical_block_size)

		// split the mountinfo line by the separator hyphen
		parts := strings.Split(line, " - ")
		if len(parts) != 2 {
			return nil, fmt.Errorf("found invalid mountinfo line in file %s: %s", mountinfo, line)
		}

		fields := strings.Fields(parts[0])
		blockDeviceID := fields[2]
		mountPoint := fields[4]
		mountOpts := fields[5]

		fields = strings.Fields(parts[1])
		fstype := fields[0]
		device := fields[1]
		blockIds := strings.Split(blockDeviceID, ":")
		majorId, err := strconv.Atoi(blockIds[0])
		if err != nil {
			return nil, err
		}
		minorId, err := strconv.Atoi(blockIds[1])
		if err != nil {
			return nil, err
		}
		d = PartitionStat{
			Device:     device,
			Mountpoint: mountPoint,
			Options:    mountOpts,
			Fstype:     fstype,
			Major:      majorId,
			Minor:      minorId,
		}
		ret = append(ret, d)
	}
	return ret, nil
}
