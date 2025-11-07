#!/usr/bin/env bash

log_error() {
	if [ -n "${PLAIN_OUTPUT}" ]
	then
		echo -e "$1"
	else
		echo -e "\e[31m$1\e[0m"
	fi
}

log_info() {
	if [ -n "${PLAIN_OUTPUT}" ]
	then
		echo -e "$1"
	else
		echo -e "\e[34m$1\e[0m"
	fi
}

log_success() {
	if [ -n "${PLAIN_OUTPUT}" ]
	then
		echo -e "$1"
	else
		echo -e "\e[32m$1\e[0m"
	fi
}

declare -A scenarioLines

if [ -n "${EXPECTED_FAILURES_FILE}" ]
then
	if [ -f "${EXPECTED_FAILURES_FILE}" ]
	then
		log_info "Checking expected failures in ${EXPECTED_FAILURES_FILE}"
	else
		log_error "Expected failures file ${EXPECTED_FAILURES_FILE} not found"
		log_error "Check the setting of EXPECTED_FAILURES_FILE environment variable"
		exit 1
	fi
	FINAL_EXIT_STATUS=0
	# If the last line of the expected-failures file ends without a newline character
	# then that line may not get processed by some of the bash code in this script
	# So check that the last character in the file is a newline
	if [ "$(tail -c1 "${EXPECTED_FAILURES_FILE}" | wc -l)" -eq 0 ]
	then
		log_error "Expected failures file ${EXPECTED_FAILURES_FILE} must end with a newline"
		log_error "Put a newline at the end of the last line and try again"
		FINAL_EXIT_STATUS=1
	fi
	# Check the expected-failures file to ensure that the lines are self-consistent
	LINE_NUMBER=0
	while read -r INPUT_LINE
		do
			LINE_NUMBER=$(("$LINE_NUMBER" + 1))

			# Ignore comment lines (starting with hash)
			if [[ "${INPUT_LINE}" =~ ^# ]]
			then
				continue
			fi
			# Match lines that have "- [someSuite/someName.feature:n]" pattern on start
			# the part inside the brackets is the suite, feature and line number of the expected failure.
			if [[ "${INPUT_LINE}" =~ ^-[[:space:]]\[([a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+\.feature:[0-9]+)] ]]; then
				SUITE_SCENARIO_LINE="${BASH_REMATCH[1]}"
			elif [[
				# report for lines like: " - someSuite/someName.feature:n"
				"${INPUT_LINE}" =~ ^[[:space:]]*-[[:space:]][a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+\.feature:[0-9]+[[:space:]]*$ ||
				# report for lines starting with: "[someSuite/someName.feature:n]"
				"${INPUT_LINE}" =~ ^[[:space:]]*\[([a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+\.feature:[0-9]+)]
			]]; then
				log_error "> Line ${LINE_NUMBER}: Not in the correct format."
				log_error "  + Actual Line     : '${INPUT_LINE}'"
				log_error "  - Expected Format : '- [suite/scenario.feature:line_number](scenario_line_url)'"
				FINAL_EXIT_STATUS=1
				continue
			else
				# otherwise, ignore the line
				continue
			fi
			# Find the link in round-brackets that should be after the SUITE_SCENARIO_LINE
			if [[ "${INPUT_LINE}" =~ \(([a-zA-Z0-9:/.#_-]+)\) ]]; then
				ACTUAL_LINK="${BASH_REMATCH[1]}"
			else
				log_error "Line ${LINE_NUMBER}: ${INPUT_LINE} : Link is empty"
				FINAL_EXIT_STATUS=1
				continue
			fi
			if [[ -n "${scenarioLines[${SUITE_SCENARIO_LINE}]:-}" ]];
			then
				log_error "> Line ${LINE_NUMBER}: Scenario line ${SUITE_SCENARIO_LINE} is duplicated"
				FINAL_EXIT_STATUS=1
			fi
			scenarioLines[${SUITE_SCENARIO_LINE}]="exists"
			OLD_IFS=${IFS}
			IFS=':'
			read -ra FEATURE_PARTS <<< "${SUITE_SCENARIO_LINE}"
			IFS=${OLD_IFS}
			SUITE_FEATURE="${FEATURE_PARTS[0]}"
			FEATURE_LINE="${FEATURE_PARTS[1]}"
			EXPECTED_LINK="https://github.com/opencloud-eu/opencloud/blob/main/tests/acceptance/features/${SUITE_FEATURE}#L${FEATURE_LINE}"
			if [[ "${ACTUAL_LINK}" != "${EXPECTED_LINK}" ]]; then
				log_error "> Line ${LINE_NUMBER}: Link is not correct for ${SUITE_SCENARIO_LINE}"
				log_error "  + Actual link   : ${ACTUAL_LINK}"
				log_error "  - Expected link : ${EXPECTED_LINK}"
				FINAL_EXIT_STATUS=1
			fi

		done < "${EXPECTED_FAILURES_FILE}"
else
	log_error "Environment variable EXPECTED_FAILURES_FILE must be defined to be the file to check"
	exit 1
fi

if [ ${FINAL_EXIT_STATUS} == 1 ]
then
	log_error "\nErrors were found in the expected failures file - see the messages above!"
else
	log_success "\nNo problems were found in the expected failures file."
fi
exit ${FINAL_EXIT_STATUS}
