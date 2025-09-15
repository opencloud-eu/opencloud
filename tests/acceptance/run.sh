#!/usr/bin/env bash
[[ "${DEBUG}" == "true" ]] && set -x

# from http://stackoverflow.com/a/630387
SCRIPT_PATH="`dirname \"$0\"`" # relative
SCRIPT_PATH="`( cd \"${SCRIPT_PATH}\" && pwd )`" # absolutized and normalized

echo 'Script path: '${SCRIPT_PATH}

# Allow optionally passing in the path to the behat program.
# This gives flexibility for callers that have installed their own behat
if [ -z "${BEHAT_BIN}" ]
then
	BEHAT=${SCRIPT_PATH}/../../vendor-bin/behat/vendor/bin/behat
else
	BEHAT=${BEHAT_BIN}
fi
BEHAT_TAGS_OPTION_FOUND=false

if [ -n "${STEP_THROUGH}" ]
then
	STEP_THROUGH_OPTION="--step-through"
fi

if [ -n "${STOP_ON_FAILURE}" ]
then
	STOP_OPTION="--stop-on-failure"
fi

if [ -n "${PLAIN_OUTPUT}" ]
then
	# explicitly tell Behat to not do colored output
	COLORS_OPTION="--no-colors"
	# Use the Bash "null" command to do nothing, rather than use tput to set a color
	RED_COLOR=":"
	GREEN_COLOR=":"
	YELLOW_COLOR=":"
else
	COLORS_OPTION="--colors"
	RED_COLOR="tput setaf 1"
	GREEN_COLOR="tput setaf 2"
	YELLOW_COLOR="tput setaf 3"
fi

# The following environment variables can be specified:
#
# ACCEPTANCE_TEST_TYPE - see "--type" description
# BEHAT_FEATURE - see "--feature" description
# BEHAT_FILTER_TAGS - see "--tags" description
# BEHAT_SUITE - see "--suite" description
# BEHAT_YML - see "--config" description
# RUN_PART and DIVIDE_INTO_NUM_PARTS - see "--part" description
# SHOW_OC_LOGS - see "--show-oc-logs" description
# TESTING_REMOTE_SYSTEM - see "--remote" description
# EXPECTED_FAILURES_FILE - a file that contains a list of the scenarios that are expected to fail

if [ -n "${EXPECTED_FAILURES_FILE}" ]
then
	# Check the expected-failures file
	${SCRIPT_PATH}/lint-expected-failures.sh
	LINT_STATUS=$?
	if [ ${LINT_STATUS} -ne 0 ]
	then
		echo "Error: expected failures file ${EXPECTED_FAILURES_FILE} is invalid"
		exit ${LINT_STATUS}
	fi
fi

# Default to API tests
# Note: if a specific feature or suite is also specified, then the acceptance
#       test type is deduced from the suite name, and this environment variable
#       ACCEPTANCE_TEST_TYPE is overridden.
if [ -z "${ACCEPTANCE_TEST_TYPE}" ]
then
	ACCEPTANCE_TEST_TYPE="api"
fi

# Look for command line options for:
# -c or --config - specify a behat.yml to use
# --feature - specify a single feature to run
# --suite - specify a single suite to run
# --type - api or core-api - if no individual feature or suite is specified, then
#          specify the type of acceptance tests to run. Default api.
# --tags - specify tags for scenarios to run (or not)
# --remote - the server under test is remote, so we cannot locally enable the
#            testing app. We have to assume it is already enabled.
# --show-oc-logs - tail the OpenCloud log after the test run
# --loop - loop tests for given number of times. Only use it for debugging purposes
# --part - run a subset of scenarios, need two numbers.
#          first number: which part to run
#          second number: in how many parts to divide the set of scenarios
# --step-through - pause after each test step

# Command line options processed here will override environment variables that
# might have been set by the caller, or in the code above.
while [[ $# -gt 0 ]]
do
	key="$1"
	case ${key} in
		-c|--config)
			BEHAT_YML="$2"
			shift
			;;
		--feature)
			BEHAT_FEATURE="$2"
			shift
			;;
		--suite)
			BEHAT_SUITE="$2"
			shift
			;;
		--loop)
			BEHAT_RERUN_TIMES="$2"
			shift
			;;
		--type)
			# Lowercase the parameter value, so the user can provide "API", "CORE-API", etc
			ACCEPTANCE_TEST_TYPE="${2,,}"
			shift
			;;
		--tags)
			BEHAT_FILTER_TAGS="$2"
			BEHAT_TAGS_OPTION_FOUND=true
			shift
			;;
		--part)
			RUN_PART="$2"
			DIVIDE_INTO_NUM_PARTS="$3"
			if [ ${RUN_PART} -gt ${DIVIDE_INTO_NUM_PARTS} ]
			then
				echo "cannot run part ${RUN_PART} of ${DIVIDE_INTO_NUM_PARTS}"
				exit 1
			fi
			shift 2
			;;
		--step-through)
			STEP_THROUGH_OPTION="--step-through"
			;;
		*)
			# A "random" parameter is presumed to be a feature file to run.
			# Typically that will be specified at the end, or as the only
			# parameter.
			BEHAT_FEATURE="$1"
			;;
	esac
	shift
done

# Set the language to "C"
# We want to have it all in english to be able to parse outputs
export LANG=C

# Provide a default admin username and password.
# But let the caller pass them if they wish
if [ -z "${ADMIN_USERNAME}" ]
then
	ADMIN_USERNAME="admin"
fi

if [ -z "${ADMIN_PASSWORD}" ]
then
	ADMIN_PASSWORD="admin"
fi

export ADMIN_USERNAME
export ADMIN_PASSWORD

if [ -z "${BEHAT_RERUN_TIMES}" ]
then
	BEHAT_RERUN_TIMES=1
fi

# expected variables
# --------------------
# $SUITE_FEATURE_TEXT - human readable which test to run
# $BEHAT_SUITE_OPTION - suite setting with "--suite" or empty if all suites have to be run
# $BEHAT_FEATURE - feature file, or empty
# $BEHAT_FILTER_TAGS - list of tags
# $BEHAT_TAGS_OPTION_FOUND
# $TEST_LOG_FILE
# $BEHAT - behat executable
# $BEHAT_YML
#
# set arrays
# ---------------
# $UNEXPECTED_FAILED_SCENARIOS array of scenarios that failed unexpectedly
# $UNEXPECTED_PASSED_SCENARIOS array of scenarios that passed unexpectedly (while running with expected-failures.txt)
# $STOP_ON_FAILURE - aborts the test run after the first failure

declare -a UNEXPECTED_FAILED_SCENARIOS
declare -a UNEXPECTED_PASSED_SCENARIOS
declare -a UNEXPECTED_BEHAT_EXIT_STATUSES

function run_behat_tests() {
	echo "Running ${SUITE_FEATURE_TEXT} tests tagged ${BEHAT_FILTER_TAGS}" | tee ${TEST_LOG_FILE}

	if [ "${REPLACE_USERNAMES}" == "true" ]
	then
		echo "Usernames and attributes in tests are being replaced:"
		cat ${SCRIPT_PATH}/usernames.json
	fi

	echo "Using behat config '${BEHAT_YML}'"
	${BEHAT} ${COLORS_OPTION} ${STOP_OPTION} --strict ${STEP_THROUGH_OPTION} -c ${BEHAT_YML} -f pretty ${BEHAT_SUITE_OPTION} --tags ${BEHAT_FILTER_TAGS} ${BEHAT_FEATURE} -v 2>&1 | tee -a ${TEST_LOG_FILE}

	BEHAT_EXIT_STATUS=${PIPESTATUS[0]}

	# remove nullbytes from the test log
	sed -i 's/\x0//g' ${TEST_LOG_FILE}

	# Find the count of scenarios that passed
	SCENARIO_RESULTS_COLORED=`grep -Ea '^[0-9]+[[:space:]]scenario(|s)[[:space:]]\(' ${TEST_LOG_FILE}`
	SCENARIO_RESULTS=$(echo "${SCENARIO_RESULTS_COLORED}" | sed "s/\x1b[^m]*m//g")
	if [ ${BEHAT_EXIT_STATUS} -eq 0 ]
	then
		# They (SCENARIO_RESULTS) all passed, so just get the first number.
		# The text looks like "1 scenario (1 passed)" or "123 scenarios (123 passed)"
		[[ ${SCENARIO_RESULTS} =~ ([0-9]+) ]]
		SCENARIOS_THAT_PASSED=$((SCENARIOS_THAT_PASSED + BASH_REMATCH[1]))
	else
		# "Something went wrong" with the Behat run (non-zero exit status).
		# If there were "ordinary" test fails, then we process that later. Maybe they are all "expected failures".
		# But if there were steps in a feature file that are undefined, we want to fail immediately.
		# So exit the tests and do not lint expected failures when undefined steps exist.
		if [[ ${SCENARIO_RESULTS} == *"undefined"* ]]
		then
			${RED_COLOR}; echo -e "Undefined steps: There were some undefined steps found."
			exit 1
		fi
		# If there were no scenarios in the requested suite or feature that match
		# the requested combination of tags, then Behat exits with an error status
		# and reports "No scenarios" in its output.
		# This can happen, for example, when running core suites from an app and
		# requesting some tag combination that does not happen frequently. Then
		# sometimes there may not be any matching scenarios in one of the suites.
		# In this case, consider the test has passed.
		MATCHING_COUNT=`grep -ca '^No scenarios$' ${TEST_LOG_FILE}`
		if [ ${MATCHING_COUNT} -eq 1 ]
		then
			echo "Information: no matching scenarios were found."
			BEHAT_EXIT_STATUS=0
		else
			# Find the count of scenarios that passed and failed
			SCENARIO_RESULTS_COLORED=`grep -Ea '^[0-9]+[[:space:]]scenario(|s)[[:space:]]\(' ${TEST_LOG_FILE}`
			SCENARIO_RESULTS=$(echo "${SCENARIO_RESULTS_COLORED}" | sed "s/\x1b[^m]*m//g")
			if [[ ${SCENARIO_RESULTS} =~ [0-9]+[^0-9]+([0-9]+)[^0-9]+([0-9]+)[^0-9]+ ]]
			then
				# Some passed and some failed, we got the second and third numbers.
				# The text looked like "15 scenarios (6 passed, 9 failed)"
				SCENARIOS_THAT_PASSED=$((SCENARIOS_THAT_PASSED + BASH_REMATCH[1]))
				SCENARIOS_THAT_FAILED=$((SCENARIOS_THAT_FAILED + BASH_REMATCH[2]))
			elif [[ ${SCENARIO_RESULTS} =~ [0-9]+[^0-9]+([0-9]+)[^0-9]+ ]]
			then
				# All failed, we got the second number.
				# The text looked like "4 scenarios (4 failed)"
				SCENARIOS_THAT_FAILED=$((SCENARIOS_THAT_FAILED + BASH_REMATCH[1]))
			fi
		fi
	fi

	FAILED_SCENARIO_PATHS_COLORED=`awk '/Failed scenarios:/',0 ${TEST_LOG_FILE} | grep -a feature`
	# There will be some ANSI escape codes for color in the FEATURE_COLORED var.
	# Strip them out so we can pass just the ordinary feature details to Behat.
	# Also strip everything after ".feature:XX", including text such as "(on line xx)" added by Behat indicating the failing step's line number.
	# Thanks to https://en.wikipedia.org/wiki/Tee_(command) and
	# https://stackoverflow.com/questions/23416278/how-to-strip-ansi-escape-sequences-from-a-variable
	# for ideas.
	FAILED_SCENARIO_PATHS=$(echo "${FAILED_SCENARIO_PATHS_COLORED}" | sed "s/\x1b[^m]*m//g" | sed 's/\(\.feature:[0-9]\+\).*/\1/')

	# If something else went wrong, and there were no failed scenarios,
	# then the awk, grep, sed command sequence above ends up with an empty string.
	# Unset FAILED_SCENARIO_PATHS to avoid later code thinking that there might be
	# one failed scenario.
	if [ -z "${FAILED_SCENARIO_PATHS}" ]
	then
		unset FAILED_SCENARIO_PATHS
	fi

	if [ -n "${EXPECTED_FAILURES_FILE}" ]
	then
		if [ -n "${BEHAT_SUITE_TO_RUN}" ]
		then
			echo "Checking expected failures for suite ${BEHAT_SUITE_TO_RUN}"
		else
			echo "Checking expected failures"
		fi

		# Check that every failed scenario is in the list of expected failures
		for FAILED_SCENARIO_PATH in ${FAILED_SCENARIO_PATHS}
			do
				SUITE_PATH=`dirname ${FAILED_SCENARIO_PATH}`
				SUITE=`basename ${SUITE_PATH}`
				SCENARIO=`basename ${FAILED_SCENARIO_PATH}`
				SUITE_SCENARIO="${SUITE}/${SCENARIO}"
				grep "\[${SUITE_SCENARIO}\]" "${EXPECTED_FAILURES_FILE}" > /dev/null
				if [ $? -ne 0 ]
				then
					echo "Error: Scenario ${SUITE_SCENARIO} failed but was not expected to fail."
					UNEXPECTED_FAILED_SCENARIOS+=("${SUITE_SCENARIO}")
				fi
			done

		# Check that every scenario in the list of expected failures did fail
		while read SUITE_SCENARIO
			do
				# Ignore comment lines (starting with hash)
				if [[ "${SUITE_SCENARIO}" =~ ^# ]]
				then
					continue
				fi
				# Match lines that have [someSuite/someName.feature:n] - the part inside the
				# brackets is the suite, feature and line number of the expected failure.
				# Else ignore the line.
				if [[ "${SUITE_SCENARIO}" =~ \[([a-zA-Z0-9-]+/[a-zA-Z0-9-]+\.feature:[0-9]+)] ]]; then
					SUITE_SCENARIO="${BASH_REMATCH[1]}"
				else
					continue
				fi
				if [ -n "${BEHAT_SUITE_TO_RUN}" ]
				then
					# If the expected failure is not in the suite that is currently being run,
					# then do not try and check that it failed.
					REGEX_TO_MATCH="^${BEHAT_SUITE_TO_RUN}/"
					if ! [[ "${SUITE_SCENARIO}" =~ ${REGEX_TO_MATCH} ]]
					then
						continue
					fi
				fi

				# look for the expected suite-scenario at the end of a line in the
				# FAILED_SCENARIO_PATHS - for example looking for apiComments/comments.feature:9
				# we want to match lines like:
				# tests/acceptance/features/apiComments/comments.feature:9
				# but not lines like::
				# tests/acceptance/features/apiComments/comments.feature:902
				echo "${FAILED_SCENARIO_PATHS}" | grep ${SUITE_SCENARIO}$ > /dev/null
				if [ $? -ne 0 ]
				then
					echo "Info: Scenario ${SUITE_SCENARIO} was expected to fail but did not fail."
					UNEXPECTED_PASSED_SCENARIOS+=("${SUITE_SCENARIO}")
				fi
			done < ${EXPECTED_FAILURES_FILE}
	else
		for FAILED_SCENARIO_PATH in ${FAILED_SCENARIO_PATHS}
		do
			SUITE_PATH=$(dirname "${FAILED_SCENARIO_PATH}")
			SUITE=$(basename "${SUITE_PATH}")
			SCENARIO=$(basename "${FAILED_SCENARIO_PATH}")
			SUITE_SCENARIO="${SUITE}/${SCENARIO}"
			UNEXPECTED_FAILED_SCENARIOS+=("${SUITE_SCENARIO}")
		done
	fi

	if [ ${BEHAT_EXIT_STATUS} -ne 0 ] && [ ${#FAILED_SCENARIO_PATHS[@]} -eq 0 ]
	then
		# Behat had some problem and there were no failed scenarios reported
		# So the problem is something else.
		# Possibly there were missing step definitions. Or Behat crashed badly, or...
		UNEXPECTED_BEHAT_EXIT_STATUSES+=("${SUITE_FEATURE_TEXT} had behat exit status ${BEHAT_EXIT_STATUS}")
	fi

	if [ "${BEHAT_TAGS_OPTION_FOUND}" != true ]
	then
		# The behat run specified to skip scenarios tagged @skip
		# Report them in a dry-run so they can be seen
		# Big red error output is displayed if there are no matching scenarios - send it to null
		DRY_RUN_FILE=$(mktemp)
		SKIP_TAGS="@skip"
		${BEHAT} --dry-run {$COLORS_OPTION} -c ${BEHAT_YML} -f pretty ${BEHAT_SUITE_OPTION} --tags "${SKIP_TAGS}" ${BEHAT_FEATURE} 1>${DRY_RUN_FILE} 2>/dev/null
		if grep -q -m 1 'No scenarios' "${DRY_RUN_FILE}"
		then
			# If there are no skip scenarios, then no need to report that
			:
		else
			echo ""
			echo "The following tests were skipped because they are tagged @skip:"
			cat "${DRY_RUN_FILE}" | tee -a ${TEST_LOG_FILE}
		fi
		rm -f "${DRY_RUN_FILE}"
	fi
}

declare -x TEST_SERVER_URL

if [ -z "${IPV4_URL}" ]
then
	IPV4_URL="${TEST_SERVER_URL}"
fi

if [ -z "${IPV6_URL}" ]
then
	IPV6_URL="${TEST_SERVER_URL}"
fi

# If a feature file has been specified but no suite, then deduce the suite
if [ -n "${BEHAT_FEATURE}" ] && [ -z "${BEHAT_SUITE}" ]
then
	SUITE_PATH=`dirname ${BEHAT_FEATURE}`
	BEHAT_SUITE=`basename ${SUITE_PATH}`
fi

if [ -z "${BEHAT_YML}" ]
then
	# Look for a behat.yml somewhere below the current working directory
	# This saves app acceptance tests being forced to specify BEHAT_YML
	BEHAT_YML="config/behat.yml"
	if [ ! -f "${BEHAT_YML}" ]
	then
		BEHAT_YML="acceptance/config/behat.yml"
	fi
	if [ ! -f "${BEHAT_YML}" ]
	then
		BEHAT_YML="tests/acceptance/config/behat.yml"
	fi
	# If no luck above, then use the core behat.yml that should live below this script
	if [ ! -f "${BEHAT_YML}" ]
	then
		BEHAT_YML="${SCRIPT_PATH}/config/behat.yml"
	fi
fi

BEHAT_CONFIG_DIR=$(dirname "${BEHAT_YML}")
ACCEPTANCE_DIR=$(dirname "${BEHAT_CONFIG_DIR}")
BEHAT_FEATURES_DIR="${ACCEPTANCE_DIR}/features"

declare -a BEHAT_SUITES

function get_behat_suites() {
	# $1 type of suites to get "api" or "core-api"
	# defaults to "api"
	TYPE="$1"
	if [[ -z "$TYPE" ]]
	then
		TYPE="api"
	fi
	ALL_SUITES=`find ${BEHAT_FEATURES_DIR}/ -type d -iname ${TYPE}* | sort | rev | cut -d"/" -f1 | rev`
	COUNT_ALL_SUITES=`echo "${ALL_SUITES}" | wc -l`
	#divide the suites letting it round down (could be zero)
	MIN_SUITES_PER_RUN=$((${COUNT_ALL_SUITES} / ${DIVIDE_INTO_NUM_PARTS}))
	#some jobs might need an extra suite
	MAX_SUITES_PER_RUN=$((${MIN_SUITES_PER_RUN} + 1))
	# the remaining number of suites that need to be distributed (could be zero)
	REMAINING_SUITES=$((${COUNT_ALL_SUITES} - (${DIVIDE_INTO_NUM_PARTS} * ${MIN_SUITES_PER_RUN})))

	if [[ ${RUN_PART} -le ${REMAINING_SUITES} ]]
	then
		SUITES_THIS_RUN=${MAX_SUITES_PER_RUN}
		SUITES_IN_PREVIOUS_RUNS=$((${MAX_SUITES_PER_RUN} * (${RUN_PART} - 1)))
	else
		SUITES_THIS_RUN=${MIN_SUITES_PER_RUN}
		SUITES_IN_PREVIOUS_RUNS=$((((${MAX_SUITES_PER_RUN} * ${REMAINING_SUITES}) + (${MIN_SUITES_PER_RUN} * (${RUN_PART} - ${REMAINING_SUITES} - 1)))))
	fi

	if [ ${SUITES_THIS_RUN} -eq 0 ]
	then
		echo "there are only ${COUNT_ALL_SUITES} suites, nothing to do in part ${RUN_PART}"
		exit 0
	fi

	COUNT_FINISH_AND_TODO_SUITES=$((${SUITES_IN_PREVIOUS_RUNS} + ${SUITES_THIS_RUN}))
	BEHAT_SUITES+=(`echo "${ALL_SUITES}" | head -n ${COUNT_FINISH_AND_TODO_SUITES} | tail -n ${SUITES_THIS_RUN}`)
}

if [[ -n "${BEHAT_SUITE}" ]]
then
	BEHAT_SUITES+=("${BEHAT_SUITE}")
else
	if [[ -n "${RUN_PART}" ]]; then
		if [[ "${ACCEPTANCE_TEST_TYPE}" == "core-api" ]]; then
			get_behat_suites "core"
		else
			get_behat_suites "${ACCEPTANCE_TEST_TYPE}"
		fi
	else
	  BEHAT_SUITES=(`echo "$BEHAT_SUITES" | tr "," "\n"`)
	fi
fi


TEST_TYPE_TEXT="API"

# Always have "@api"
if [ ! -z "${BEHAT_FILTER_TAGS}" ]
then
	# Be nice to the caller
	# Remove any extra "&&" at the end of their tags list
	BEHAT_FILTER_TAGS="${BEHAT_FILTER_TAGS%&&}"
	# Remove any extra "&&" at the beginning of their tags list
	BEHAT_FILTER_TAGS="${BEHAT_FILTER_TAGS#&&}"
fi

# EMAIL_HOST defines where the system-under-test can find the email server (inbucket)
# for sending email.
if [ -z "${EMAIL_HOST}" ]
then
	EMAIL_HOST="127.0.0.1"
fi

# LOCAL_INBUCKET_HOST defines where this test script can find the Inbucket server
# for sending email. When testing a remote system, the Inbucket server somewhere
# "in the middle" might have a different host name from the point of view of
# the test script.
if [ -z "${LOCAL_EMAIL_HOST}" ]
then
	LOCAL_EMAIL_HOST="${EMAIL_HOST}"
fi

if [ -z "${EMAIL_SMTP_PORT}" ]
then
	EMAIL_SMTP_PORT="2500"
fi

# If the caller did not mention specific tags, skip the skipped tests by default
if [ "${BEHAT_TAGS_OPTION_FOUND}" = false ]
then
	if [[ -z $BEHAT_FILTER_TAGS ]]
	then
		BEHAT_FILTER_TAGS="~@skip"
	# If the caller has already specified specifically to run "@skip" scenarios
	# then do not append "not @skip"
	elif [[ ! ${BEHAT_FILTER_TAGS} =~ (^|&)@skip(&|$) ]]
	then
		BEHAT_FILTER_TAGS="${BEHAT_FILTER_TAGS}&&~@skip"
	fi
fi

export IPV4_URL
export IPV6_URL
export FILES_FOR_UPLOAD="${SCRIPT_PATH}/filesForUpload/"

TEST_LOG_FILE=$(mktemp)
SCENARIOS_THAT_PASSED=0
SCENARIOS_THAT_FAILED=0

if [ ${#BEHAT_SUITES[@]} -eq 0 ] && [ -z "${BEHAT_FEATURE}" ]
then
	SUITE_FEATURE_TEXT="all ${TEST_TYPE_TEXT}"
	run_behat_tests
else
	if [ -n "${BEHAT_SUITE}" ]
	then
		SUITE_FEATURE_TEXT="${BEHAT_SUITE}"
	fi

	if [ -n "${BEHAT_FEATURE}" ]
	then
		# If running a whole feature, it will be something like login.feature
		# If running just a single scenario, it will also have the line number
		# like login.feature:36 - which will be parsed correctly like a "file"
		# by basename.
		BEHAT_FEATURE_FILE=`basename ${BEHAT_FEATURE}`
		SUITE_FEATURE_TEXT="${SUITE_FEATURE_TEXT} ${BEHAT_FEATURE_FILE}"
	fi
fi

for i in "${!BEHAT_SUITES[@]}"
	do
		BEHAT_SUITE_TO_RUN="${BEHAT_SUITES[$i]}"
		BEHAT_SUITE_OPTION="--suite=${BEHAT_SUITE_TO_RUN}"
		SUITE_FEATURE_TEXT="${BEHAT_SUITES[$i]}"
		for rerun_number in $(seq 1 ${BEHAT_RERUN_TIMES})
			do
				if ((${BEHAT_RERUN_TIMES} > 1))
				then
					echo -e "\nTest repeat $rerun_number of ${BEHAT_RERUN_TIMES}"
				fi
				run_behat_tests
			done
done

TOTAL_SCENARIOS=$((SCENARIOS_THAT_PASSED + SCENARIOS_THAT_FAILED))

echo "runsh: Total ${TOTAL_SCENARIOS} scenarios (${SCENARIOS_THAT_PASSED} passed, ${SCENARIOS_THAT_FAILED} failed)"

# 3 types of things can have gone wrong:
#   - some scenario failed (and it was not expected to fail)
#   - some scenario passed (but it was expected to fail)
#   - Behat exited with non-zero status because of some other error
# If any of these happened then report about it and exit with status 1 (error)

if [ ${#UNEXPECTED_FAILED_SCENARIOS[@]} -gt 0 ]
then
	UNEXPECTED_FAILURE=true
else
	UNEXPECTED_FAILURE=false
fi

if [ ${#UNEXPECTED_PASSED_SCENARIOS[@]} -gt 0 ]
then
	UNEXPECTED_SUCCESS=true
else
	UNEXPECTED_SUCCESS=false
fi

if [ ${#UNEXPECTED_BEHAT_EXIT_STATUSES[@]} -gt 0 ]
then
	UNEXPECTED_BEHAT_EXIT_STATUS=true
else
	UNEXPECTED_BEHAT_EXIT_STATUS=false
fi

# If we got some unexpected success, and we only ran a single feature or scenario
# then the fact that some expected failures did not happen might be because those
# scenarios were never even run.
# Filter the UNEXPECTED_PASSED_SCENARIOS to remove scenarios that were not run.
if [ "${UNEXPECTED_SUCCESS}" = true ]
then
	ACTUAL_UNEXPECTED_PASS=()
	# if running a single feature or a single scenario
	if [[ -n "${BEHAT_FEATURE}" ]]
	then
		for unexpected_passed_value in "${UNEXPECTED_PASSED_SCENARIOS[@]}"
		do
			# check only for the running feature
			if [[ $BEHAT_FEATURE == *":"* ]]
			then
				BEHAT_FEATURE_WITH_LINE_NUM=$BEHAT_FEATURE
			else
				LINE_NUM=$(echo ${unexpected_passed_value} | cut -d":" -f2)
				BEHAT_FEATURE_WITH_LINE_NUM=$BEHAT_FEATURE:$LINE_NUM
			fi
			if [[ $BEHAT_FEATURE_WITH_LINE_NUM == *"${unexpected_passed_value}" ]]
			then
				ACTUAL_UNEXPECTED_PASS+=("${unexpected_passed_value}")
			fi
		done
	else
		ACTUAL_UNEXPECTED_PASS=("${UNEXPECTED_PASSED_SCENARIOS[@]}")
	fi

	if [ ${#ACTUAL_UNEXPECTED_PASS[@]} -eq 0 ]
	then
		UNEXPECTED_SUCCESS=false
	fi
fi

if [ "${UNEXPECTED_FAILURE}" = false ] && [ "${UNEXPECTED_SUCCESS}" = false ] && [ "${UNEXPECTED_BEHAT_EXIT_STATUS}" = false ]
then
	FINAL_EXIT_STATUS=0
else
	FINAL_EXIT_STATUS=1
fi

if [ -n "${EXPECTED_FAILURES_FILE}" ]
then
	echo "runsh: Exit code after checking expected failures: ${FINAL_EXIT_STATUS}"
fi

if [ "${UNEXPECTED_FAILURE}" = true ]
then
	${YELLOW_COLOR}; echo "runsh: Total unexpected failed scenarios throughout the test run:"
	${RED_COLOR}; printf "%s\n" "${UNEXPECTED_FAILED_SCENARIOS[@]}"
else
	${GREEN_COLOR}; echo "runsh: There were no unexpected failures."
fi

if [ "${UNEXPECTED_SUCCESS}" = true ]
then
	${YELLOW_COLOR}; echo "runsh: Total unexpected passed scenarios throughout the test run:"
	${RED_COLOR}; printf "%s\n" "${ACTUAL_UNEXPECTED_PASS[@]}"
else
	${GREEN_COLOR}; echo "runsh: There were no unexpected success."
fi

if [ "${UNEXPECTED_BEHAT_EXIT_STATUS}" = true ]
then
	${YELLOW_COLOR}; echo "runsh: The following Behat test runs exited with non-zero status:"
	${RED_COLOR}; printf "%s\n" "${UNEXPECTED_BEHAT_EXIT_STATUSES[@]}"
fi

# # sync the file-system so all output will be flushed to storage.
# # In drone we sometimes see that the last lines of output are missing from the
# # drone log.
# sync

# # If we are running in drone CI, then sleep for a bit to (hopefully) let the
# # drone agent send all the output to the drone server.
# if [ -n "${CI_REPO}" ]
# then
# 	echo "sleeping for 30 seconds at end of test run"
# 	sleep 30
# fi

exit ${FINAL_EXIT_STATUS}
