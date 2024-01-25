package main

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
	"syscall"
	"time"
)

var (
	longListing   bool
	recursive     bool
	allFiles      bool
	reverse       bool
	sortByModTime bool
)

func main() {
	parseFlags()

	path := "." + string(os.PathSeparator)
	if len(os.Args) > 1 {
		if os.Args[1] != "-l" {
			path = os.Args[1]
		}
	}

	err := listFiles(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing files: %v\n", err)
		os.Exit(1)
	}
}

func parseFlags() {
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-l":
			longListing = true
		case "-R":
			recursive = true
		case "-a":
			allFiles = true
		case "-r":
			reverse = true
		case "-t":
			sortByModTime = true
		default:
			// Ignore non-flag arguments
		}
	}
}

func listFiles(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()

	entries, err := dir.Readdirnames(-1)
	if err != nil {
		return err
	}

	if allFiles {
		hiddenFiles, err := getHiddenFiles(path)
		if err != nil {
			return err
		}
		entries = append(entries, hiddenFiles...)
	}

	if sortByModTime {
		sortSliceByModTime(entries, path)
	} else if reverse {
		sortSliceReverse(entries)
	}

	for _, entry := range entries {
		listFileDetails(path, entry)

		if recursive {
			subPath := path + string(os.PathSeparator) + entry
			subInfo, err := os.Stat(subPath)
			if err != nil {
				return err
			}
			if subInfo.IsDir() {
				fmt.Printf("\n%s:\n", subPath)
				err := listFiles(subPath)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func getHiddenFiles(path string) ([]string, error) {
	dir, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer dir.Close()

	allEntries, err := dir.Readdirnames(-1)
	if err != nil {
		return nil, err
	}

	var hiddenFiles []string
	for _, entry := range allEntries {
		if entry[0] == '.' {
			hiddenFiles = append(hiddenFiles, entry)
		}
	}

	return hiddenFiles, nil
}

func getFileModTime(filePath string) (time.Time, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return time.Time{}, err
	}
	sys := fileInfo.Sys().(*syscall.Stat_t)
	return time.Unix(sys.Mtim.Sec, sys.Mtim.Nsec), nil
}

func sortSliceByModTime(slice []string, path string) {
	customSort(slice, func(i, j int) bool {
		timeI, errI := getFileModTime(path + string(os.PathSeparator) + slice[i])
		timeJ, errJ := getFileModTime(path + string(os.PathSeparator) + slice[j])

		if errI != nil || errJ != nil {
			return slice[i] < slice[j]
		}

		if reverse {
			return timeI.After(timeJ)
		}
		return timeI.Before(timeJ)
	})
}

func sortSliceReverse(slice []string) {
	customSort(slice, func(i, j int) bool {
		return slice[j] < slice[i]
	})
}

func customSort(slice []string, less func(i, j int) bool) {
	n := len(slice)
	for i := 0; i < n-1; i++ {
		minIndex := i
		for j := i + 1; j < n; j++ {
			if less(j, minIndex) {
				minIndex = j
			}
		}
		if minIndex != i {
			slice[i], slice[minIndex] = slice[minIndex], slice[i]
		}
	}
}

func listFileDetails(path, entry string) {
	fileInfo, err := os.Stat(path + string(os.PathSeparator) + entry)
	if err != nil {
		fmt.Println(err)
		return
	}

	mode := fileInfo.Mode()
	uid := int(fileInfo.Sys().(*syscall.Stat_t).Uid)
	gid := int(fileInfo.Sys().(*syscall.Stat_t).Gid)
	size := fileInfo.Size()
	modTime := fileInfo.ModTime().Format("Jan _2 15:04")
	name := fileInfo.Name()

	permissions := getPermissions(mode)
	owner := getOwner(uid)
	group := getGroup(gid)

	fmt.Printf("%s %d %s %s %d %s %s\n", permissions, uid, owner, group, size, modTime, name)
}

func getPermissions(mode os.FileMode) string {
	const (
		ownerRead  = 0400
		ownerWrite = 0200
		ownerExec  = 0100
		groupRead  = 0040
		groupWrite = 0020
		groupExec  = 0010
		otherRead  = 0004
		otherWrite = 0002
		otherExec  = 0001
	)

	perms := "---------"
	if mode&ownerRead != 0 {
		perms = setCharAt(perms, 0, 'r')
	}
	if mode&ownerWrite != 0 {
		perms = setCharAt(perms, 1, 'w')
	}
	if mode&ownerExec != 0 {
		perms = setCharAt(perms, 2, 'x')
	}
	if mode&groupRead != 0 {
		perms = setCharAt(perms, 3, 'r')
	}
	if mode&groupWrite != 0 {
		perms = setCharAt(perms, 4, 'w')
	}
	if mode&groupExec != 0 {
		perms = setCharAt(perms, 5, 'x')
	}
	if mode&otherRead != 0 {
		perms = setCharAt(perms, 6, 'r')
	}
	if mode&otherWrite != 0 {
		perms = setCharAt(perms, 7, 'w')
	}
	if mode&otherExec != 0 {
		perms = setCharAt(perms, 8, 'x')
	}

	return perms
}

func setCharAt(str string, index int, char byte) string {
	if index < 0 || index >= len(str) {
		return str
	}
	return str[:index] + string(char) + str[index+1:]
}

func getOwner(uid int) string {
	user, err := user.LookupId(strconv.Itoa(uid))
	if err != nil {
		return strconv.Itoa(uid)
	}
	return user.Username
}

func getGroup(gid int) string {
	group, err := user.LookupGroupId(strconv.Itoa(gid))
	if err != nil {
		return strconv.Itoa(gid)
	}
	return group.Name
}
