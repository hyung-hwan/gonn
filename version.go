package gonn

const Revision string = "0.7.3"

func Release() bool {
	return Revision != "" && !strings_contains(Revision, "-")
}

func Version() string {
	var parts []string
	var number string
	var numbers []int
	var index int
	var extra int
	var value int

	number = revision_prefix(Revision)
	parts = split_revision_parts(number)
	numbers = make([]int, 0, len(parts))

	for index = 0; index < len(parts); index++ {
		value = atoi_or_zero(parts[index])
		numbers = append(numbers, value)
	}

	for len(numbers) > 3 {
		extra = numbers[len(numbers) - 1]
		numbers = numbers[:len(numbers) - 1]
		numbers[2] = numbers[2] + extra
	}

	return join_version_parts(numbers)
}

func revision_prefix(revision string) string {
	var index int
	var ch byte

	for index = 0; index < len(revision); index++ {
		ch = revision[index]
		if (ch < '0' || ch > '9') && ch != '.' && ch != '-' {
			return revision[:index]
		}
	}

	return revision
}

func split_revision_parts(number string) []string {
	var parts []string

	parts = split_fields(number, ".-")
	return parts
}

func join_version_parts(parts []int) string {
	var values []string
	var index int

	values = make([]string, 0, len(parts))

	for index = 0; index < len(parts); index++ {
		values = append(values, itoa(parts[index]))
	}

	return join_strings(values, ".")
}
