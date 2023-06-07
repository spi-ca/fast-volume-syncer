package common

type RsyncArgs struct {
	Verbose            bool
	PreservePermission bool
	PreserveOwnership  bool
	CopySpecial        bool
	Compress           bool
	WholeFile          bool
	Inplace            bool
	Recursive          bool
}

func (a *RsyncArgs) Assemble(src, dst string) []string {
	args := []string{
		"--links",
		"--hard-links",
		"--copy-dirlinks",
		"--times",
		"--omit-dir-times",
		"--omit-link-times",
		"--human-readable",
		"--protect-args",
		"--timeout=0",
		"--contimeout=0",
	}

	if a.Verbose {
		args = append(args, "--stats")
		args = append(args, "--verbose")
		args = append(args, "--progress")

	} else {
		args = append(args, "--info=NAME2")

	}

	if a.PreservePermission {
		args = append(args, "--perms")

	} else {
		args = append(args, "--no-perms")

	}

	if a.PreserveOwnership {
		args = append(args, "--owner")
		args = append(args, "--group")
	} else {
		args = append(args, "--no-owner")
		args = append(args, "--no-group")
	}

	if a.CopySpecial {
		args = append(args, "--devices")
		args = append(args, "--specials")
	} else {
		args = append(args, "--no-devices")
		args = append(args, "--no-specials")
	}

	if a.Compress {
		args = append(args, "--compress")

	} else {
		args = append(args, "--no-compress")
	}

	if a.WholeFile {
		args = append(args, "--whole-file")

	} else {
		args = append(args, "--no-whole-file")
	}

	if a.WholeFile {
		args = append(args, "--inplace")

	} else {
		args = append(args, "--no-inplace")
	}

	if a.Recursive {
		args = append(args, "--recursive")
	} else {
		args = append(args, "--no-recursive")
		args = append(args, "--files-from=-")
	}

	args = append(args, src)
	args = append(args, dst)
	return args
}
