package misc

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func RemoveFirstCharIfPresent(s string, char rune) string {
	// Check if the string is not empty and the first character matches the provided char
	if len(s) > 0 && rune(s[0]) == char {
		// Slice the string to remove the first character
		return s[1:]
	}
	return s // Return the string as is if the first character doesn't match
}

func СheckRangeFormat(s string) (bool, int, int) {
	// Define the regex pattern for the "a-b" format
	re := regexp.MustCompile(`^(\d+)-(\d+)$`)
	matches := re.FindStringSubmatch(s)

	// If no match found, return false
	if matches == nil {
		return false, 0, 0
	}

	// Parse the integers
	a, errA := strconv.Atoi(matches[1])
	b, errB := strconv.Atoi(matches[2])

	// Check for any errors during conversion
	if errA != nil || errB != nil {
		return false, 0, 0
	}

	// Ensure that a <= b
	if a >= b {
		return false, 0, 0
	}

	// Return true if the format is valid
	return true, a, b
}

// checkCommaSeparatedIntegers checks if the string consists of comma-separated integers,
// removes duplicates, and sorts the output slice.
func СheckCommaSeparatedIntegers(s string) (bool, []int) {
	// Define the regex pattern for comma-separated integers
	re := regexp.MustCompile(`^(\d+)(,\s*\d+)*$`)

	// Check if the string matches the pattern
	if !re.MatchString(s) {
		return false, nil
	}

	// Split the string by commas and parse each part into an integer
	parts := strings.Split(s, ",")
	uniqueInts := make(map[int]struct{}) // Map to store unique integers

	for _, part := range parts {
		// Trim spaces around each number
		part = strings.TrimSpace(part)
		// Convert the string part to an integer
		num, err := strconv.Atoi(part)
		if err != nil {
			// If any part is not a valid integer, return false
			return false, nil
		}
		// Add the integer to the map (duplicates are automatically removed)
		uniqueInts[num] = struct{}{}
	}

	// Convert the map keys to a slice
	var result []int
	for num := range uniqueInts {
		result = append(result, num)
	}

	// Sort the slice of integers
	sort.Ints(result)

	// Return true if all parts are valid integers, along with the sorted and deduplicated slice
	return true, result
}
