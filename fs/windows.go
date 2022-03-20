// +build windows

package fs

import (
	"github.com/hectane/go-acl/api"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"
	"os/user"
)


func GetFileOwner(path string) (string, error) {
	var (
		owner   *windows.SID
		secDesc windows.Handle
	)
	err := api.GetNamedSecurityInfo(
		path,
		api.SE_FILE_OBJECT,
		api.OWNER_SECURITY_INFORMATION,
		&owner,
		nil,
		nil,
		nil,
		&secDesc,
	)
	if err != nil {
		return "", err
	}
	defer windows.LocalFree(secDesc)

	u, err := user.LookupId(owner.String())
	if err != nil {
		return "", err
	}
	logrus.Debugf("Owner of %s is %s (%s)", path, u.Name, owner)
	return u.Name, nil
}
