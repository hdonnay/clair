// +build go1.16

package auto

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/quay/zlog"
)

type cgTestcase struct {
	Name string
	In   fstest.MapFS
	Want int
	Err  error
}

func (tc cgTestcase) Run(ctx context.Context, t *testing.T) {
	t.Run(tc.Name, func(t *testing.T) {
		ctx := zlog.Test(ctx, t)
		gmp, err := cglookup(tc.In)
		if err != tc.Err {
			t.Error(err)
		}
		if got, want := gmp, tc.Want; tc.Err == nil && got != want {
			t.Errorf("got: %v, want: %v", got, want)
		}
		Logs(ctx)
	})
}

func TestCPUDetection(t *testing.T) {
	ctx := zlog.Test(nil, t)
	t.Run("V1", func(t *testing.T) {
		const cgmap = `11:pids:/user.slice/user-1000.slice/session-4.scope
10:cpuset:/
9:blkio:/user.slice
8:hugetlb:/
7:perf_event:/
6:devices:/user.slice
5:net_cls,net_prio:/
4:cpu,cpuacct:/user.slice
3:freezer:/
2:memory:/user.slice/user-1000.slice/session-4.scope
1:name=systemd:/user.slice/user-1000.slice/session-4.scope
0::/user.slice/user-1000.slice/session-4.scope
`
		var tt = []cgTestcase{
			{
				Name: "NoLimit",
				In: fstest.MapFS{
					"proc/self/cgroup": &fstest.MapFile{Data: []byte(cgmap)},
					"sys/fs/cgroup/cpu,cpuacct/user.slice/cpu.cfs_quota_us": &fstest.MapFile{
						Data: []byte("-1\n"),
					},
				},
				Want: 0,
			},
			{
				Name: "Limit1",
				In: fstest.MapFS{
					"proc/self/cgroup": &fstest.MapFile{Data: []byte(cgmap)},
					"sys/fs/cgroup/cpu,cpuacct/user.slice/cpu.cfs_quota_us": &fstest.MapFile{
						Data: []byte("100000\n"),
					},
					"sys/fs/cgroup/cpu,cpuacct/user.slice/cpu.cfs_period_us": &fstest.MapFile{
						Data: []byte("100000\n"),
					},
				},
				Want: 1,
			},
		}
		ctx := zlog.Test(ctx, t)
		for _, tc := range tt {
			tc.Run(ctx, t)
		}
	})
	t.Run("V2", func(t *testing.T) {
		var tt = []cgTestcase{
			{
				Name: "NoLimit",
				In: fstest.MapFS{
					"proc/self/cgroup": &fstest.MapFile{
						Data: []byte("0::/\n"),
					},
					"sys/fs/cgroup/cpu.max": &fstest.MapFile{
						Data: []byte("max 100000\n"),
					},
				},
				Want: 0,
			},
			{
				Name: "Limit4",
				In: fstest.MapFS{
					"proc/self/cgroup": &fstest.MapFile{
						Data: []byte("0::/\n"),
					},
					"sys/fs/cgroup/cpu.max": &fstest.MapFile{
						Data: []byte("400000 100000\n"),
					},
				},
				Want: 4,
			},
		}
		ctx := zlog.Test(ctx, t)
		for _, tc := range tt {
			tc.Run(ctx, t)
		}
	})
}
