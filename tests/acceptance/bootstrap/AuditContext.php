<?php declare(strict_types=1);
/**
 * @copyright Copyright (c) 2026 OpenCloud GmbH <mail@opencloud.eu>
 * SPDX-License-Identifier: Apache-2.0
 */

use Behat\Behat\Context\Context;
use Behat\Behat\Hook\Scope\BeforeScenarioScope;
use Behat\Gherkin\Node\TableNode;
use Behat\Step\Then;
use PHPUnit\Framework\Assert;
use TestHelpers\BehatHelper;

require_once 'bootstrap.php';

/**
 * Checks the records of the audit log.
 *
 * The audit service writes every record as one line of JSON to stdout. The server's output goes
 * into the file AUDIT_RECORDS_FILE; lines that are not an audit record, e.g. log lines, are
 * skipped. Only the records written after the scenario started count.
 */
class AuditContext implements Context {
	private FeatureContext $featureContext;
	private string $recordsFile;
	private int $scenarioStart = 0;

	// the audit service writes a record shortly after the request, the steps wait that long for it
	private const WAIT_SECONDS = 10;
	// how long a step waits for records that must not come, or for more than the expected ones
	private const SETTLE_SECONDS = 3;

	/**
	 * @BeforeScenario
	 *
	 * @param BeforeScenarioScope $scope
	 *
	 * @return void
	 */
	public function before(BeforeScenarioScope $scope): void {
		$this->featureContext = BehatHelper::getContext($scope, $scope->getEnvironment(), 'FeatureContext');
		$this->recordsFile = (string)\getenv('AUDIT_RECORDS_FILE');
		Assert::assertNotEmpty(
			$this->recordsFile,
			'AUDIT_RECORDS_FILE must name the file the server writes its output to'
		);
		\clearstatcache(true, $this->recordsFile);
		$this->scenarioStart = \is_file($this->recordsFile) ? (int)\filesize($this->recordsFile) : 0;
	}

	/**
	 * the audit records written since the scenario started
	 *
	 * @return array<int, array<string, mixed>>
	 */
	private function records(): array {
		\clearstatcache(true, $this->recordsFile);
		if (!\is_file($this->recordsFile)) {
			return [];
		}
		$content = (string)\file_get_contents($this->recordsFile, false, null, $this->scenarioStart);
		$records = [];
		foreach (\explode("\n", $content) as $line) {
			$record = \json_decode($line, true);
			if (\is_array($record) && isset($record['event.name'], $record['schema_version'])) {
				$records[] = $record;
			}
		}
		return $records;
	}

	/**
	 * the value of a field of a record; change.before.<name> and change.after.<name> look into the
	 * changed attributes
	 *
	 * @param array<string, mixed> $record
	 * @param string $field
	 *
	 * @return string|null
	 */
	private function field(array $record, string $field): ?string {
		if (\array_key_exists($field, $record)) {
			$value = $record[$field];
		} elseif (\preg_match('/^(change\.(?:before|after))\.(.+)$/', $field, $m)
			&& isset($record[$m[1]]) && \is_array($record[$m[1]]) && \array_key_exists($m[2], $record[$m[1]])
		) {
			$value = $record[$m[1]][$m[2]];
		} else {
			return null;
		}
		if (\is_bool($value)) {
			return $value ? 'true' : 'false';
		}
		return \is_scalar($value) ? (string)$value : \json_encode($value);
	}

	/**
	 * the records of the event that have all the given field values
	 *
	 * @param string $event
	 * @param array<string, string> $fields
	 *
	 * @return array<int, array<string, mixed>>
	 */
	private function matching(string $event, array $fields): array {
		return \array_values(
			\array_filter(
				$this->records(),
				function (array $record) use ($event, $fields): bool {
					if ($record['event.name'] !== "opencloud.$event") {
						return false;
					}
					foreach ($fields as $field => $value) {
						if ($this->field($record, $field) !== $value) {
							return false;
						}
					}
					return true;
				}
			)
		);
	}

	/**
	 * the id of a user the scenario created, e.g. "Alice"
	 *
	 * @param string $user
	 *
	 * @return string
	 */
	private function userId(string $user): string {
		$id = $this->featureContext->getAttributeOfCreatedUser($user, 'id');
		Assert::assertNotEmpty($id, "the scenario did not create the user '$user'");
		return (string)$id;
	}

	/**
	 * the expected field values of a table with the columns field and value; actor.id takes the
	 * name of a user of the scenario, e.g. "Alice". actor.name is the display name of a user, or
	 * the login name of a failed login
	 *
	 * @param TableNode $table
	 *
	 * @return array<string, string>
	 */
	private function expectedFields(TableNode $table): array {
		$this->featureContext->verifyTableNodeColumns($table, ['field', 'value']);
		$fields = [];
		foreach ($table->getHash() as $row) {
			$value = $row['value'];
			if ($row['field'] === 'actor.id') {
				$value = $this->userId($value);
			}
			$fields[$row['field']] = $value;
		}
		return $fields;
	}

	/**
	 * waits until there are at least $count matching records. With $exact, it waits a moment
	 * longer to see whether more come, and checks that there are exactly $count
	 *
	 * @param int $count
	 * @param string $event
	 * @param array<string, string> $fields
	 * @param bool $exact
	 *
	 * @return void
	 */
	private function assertRecordCount(int $count, string $event, array $fields, bool $exact): void {
		$deadline = \microtime(true) + self::WAIT_SECONDS;
		while (\count($this->matching($event, $fields)) < $count && \microtime(true) < $deadline) {
			\usleep(200000);
		}
		if ($exact) {
			\sleep(self::SETTLE_SECONDS);
		}
		$found = \count($this->matching($event, $fields));
		$message = ($exact ? "expected exactly" : "expected at least")
			. " $count audit record(s) '$event' with " . \json_encode($fields) . ", found $found."
			. " Records of this scenario:\n" . \implode("\n", \array_map('json_encode', $this->records()));
		if ($exact) {
			Assert::assertSame($count, $found, $message);
		} else {
			Assert::assertGreaterThanOrEqual($count, $found, $message);
		}
	}

	/**
	 * at least one such record; the test framework itself sends requests too, e.g. a download
	 * after each upload to wait for its postprocessing
	 *
	 * @param string $event
	 * @param string $outcome
	 * @param string $user
	 *
	 * @return void
	 */
	#[Then('there should be an audit record :event with outcome :outcome by user :user')]
	public function thereShouldBeAnAuditRecordByUser(string $event, string $outcome, string $user): void {
		$this->assertRecordCount(
			1,
			$event,
			['outcome' => $outcome, 'actor.id' => $this->userId($user)],
			false
		);
	}

	/**
	 * @param string $event
	 * @param TableNode $table
	 *
	 * @return void
	 */
	#[Then('there should be an audit record :event with these fields:')]
	public function thereShouldBeAnAuditRecordWithTheseFields(string $event, TableNode $table): void {
		$this->assertRecordCount(1, $event, $this->expectedFields($table), false);
	}

	/**
	 * @param int $count
	 * @param string $event
	 * @param TableNode $table
	 *
	 * @return void
	 */
	#[Then('there should be :count audit records :event with these fields:')]
	public function thereShouldBeAuditRecordsWithTheseFields(int $count, string $event, TableNode $table): void {
		$this->assertRecordCount($count, $event, $this->expectedFields($table), true);
	}

	/**
	 * @param string $event
	 *
	 * @return void
	 */
	#[Then('there should be no audit record :event')]
	public function thereShouldBeNoAuditRecord(string $event): void {
		$this->assertRecordCount(0, $event, [], true);
	}
}
