<?php declare(strict_types=1);
/**
 * @author Sajan Gurung <sajan@jankaritech.com>
 * @copyright Copyright (c) 2024 Sajan Gurung sajan@jankaritech.com
 *
 * This code is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License,
 * as published by the Free Software Foundation;
 * either version 3 of the License, or any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <http://www.gnu.org/licenses/>
 *
 */

use Behat\Behat\Hook\Scope\BeforeScenarioScope;
use Behat\Behat\Context\Context;
use Behat\Gherkin\Node\TableNode;
use PHPUnit\Framework\Assert;
use TestHelpers\CliHelper;
use TestHelpers\OcConfigHelper;
use TestHelpers\BehatHelper;
use Psr\Http\Message\ResponseInterface;

/**
 * CLI context
 */
class CliContext implements Context {
	private FeatureContext $featureContext;
	private SpacesContext $spacesContext;

	/**
	 * opencloud users storage path
	 *
	 * @return string
	 */
	public static function getUsersStoragePath(): string {
		$path = getenv('OC_STORAGE_PATH') ?: '/var/lib/opencloud/storage/users';
		// need for CI
		$home = getenv('HOME');
		$path = preg_replace('#^~/#', $home . '/', $path);
		$path = str_replace('$HOME', $home, $path);

		return rtrim($path, '/') . '/users';
	}

	/**
	 * opencloud project spaces storage path
	 *
	 * @return string
	 */
	public static function getProjectsStoragePath(): string {
		$path = getenv('OC_STORAGE_PATH') ?: '/var/lib/opencloud/storage/users';
		return $path . '/projects';
	}

	/**
	 * @BeforeScenario
	 *
	 * @param BeforeScenarioScope $scope
	 *
	 * @return void
	 */
	public function before(BeforeScenarioScope $scope): void {
		// Get the environment
		$environment = $scope->getEnvironment();
		// Get all the contexts you need in this context
		$this->featureContext = BehatHelper::getContext($scope, $environment, 'FeatureContext');
		$this->spacesContext = BehatHelper::getContext($scope, $environment, 'SpacesContext');
	}

	/**
	 * expects a file to exist at the given path
	 *
	 * @param string $path
	 * @param string|null $sizeGb
	 * @param int $maxSeconds
	 *
	 * @return void
	 */
	private function waitForPath(string $path, ?string $sizeGb = null, int $maxSeconds = 10): void {
		$escapedPath = escapeshellarg($path);

		for ($i = 0; $i < $maxSeconds * 5; $i++) {
			if ($sizeGb) {
				if (!preg_match('/^(\d+)gb$/i', $sizeGb, $matches)) {
					throw new \InvalidArgumentException("Invalid size format: $sizeGb. Use formats like 1gb, 5gb.");
				}
				$targetBytes = (int)$matches[1] * 1024 * 1024 * 1024;
				$body = [
					"command" => "[ -f $escapedPath ] && stat -c%s $escapedPath || echo 0",
					"raw" => true
				];
				$data = json_decode((string)CliHelper::runCommand($body)->getBody(), true);

				if (isset($data['message']) && (int)trim($data['message']) >= $targetBytes) {
					return;
				}
			} else {
				$body = [
					"command" => "ls $escapedPath >/dev/null 2>&1 && echo exists || echo not_exists",
					"raw" => true
				];
				$response = CliHelper::runCommand($body);
				$data = json_decode((string)$response->getBody(), true);

				if (isset($data['message']) && trim($data['message']) === 'exists') {
					return;
				}
			}
			usleep(200000);
		}

		throw new \Exception("Timeout waiting for: $path");
	}

	/**
	 * @Given the administrator has stopped the server
	 *
	 * @return void
	 */
	public function theAdministratorHasStoppedTheServer(): void {
		$response = OcConfigHelper::stopOpencloud();
		$this->featureContext->theHTTPStatusCodeShouldBe(200, '', $response);
	}

	/**
	 * @Given /^the administrator (?:starts|has started) the server$/
	 *
	 * @return void
	 */
	public function theAdministratorHasStartedTheServer(): void {
		$response = OcConfigHelper::startOpencloud();
		$this->featureContext->theHTTPStatusCodeShouldBe(200, '', $response);
	}

	/**
	 * @When /^the administrator resets the password of (non-existing|existing) user "([^"]*)" to "([^"]*)" using the CLI$/
	 *
	 * @param string $status
	 * @param string $user
	 * @param string $password
	 *
	 * @return void
	 */
	public function theAdministratorResetsThePasswordOfUserUsingTheCLI(
		string $status,
		string $user,
		string $password
	): void {
		$command = "idm resetpassword -u $user";
		$body = [
		  "command" => $command,
		  "inputs" => [$password, $password]
		];

		$this->featureContext->setResponse(CliHelper::runCommand($body));
		if ($status === "non-existing") {
			return;
		}
		$this->featureContext->updateUserPassword($user, $password);
	}

	/**
	 * @When the administrator deletes the empty trashbin folders using the CLI
	 *
	 * @return void
	 */
	public function theAdministratorDeletesEmptyTrashbinFoldersUsingTheCli(): void {
		$path = $this->featureContext->getStorageUsersRoot();
		$command = "trash purge-empty-dirs -p $path --dry-run=false";
		$body = [
		  "command" => $command
		];
		$this->featureContext->setResponse(CliHelper::runCommand($body));
	}

	/**
	 * @When the administrator checks the backup consistency using the CLI
	 *
	 * @return void
	 */
	public function theAdministratorChecksTheBackupConsistencyUsingTheCli(): void {
		$path = $this->featureContext->getStorageUsersRoot();
		$command = "backup consistency -p $path";
		$body = [
		  "command" => $command
		];
		$this->featureContext->setResponse(CliHelper::runCommand($body));
	}

	/**
	 * @When the administrator creates app token for user :user with expiration time :expirationTime using the auth-app CLI
	 *
	 * @param string $user
	 * @param string $expirationTime
	 *
	 * @return void
	 */
	public function theAdministratorCreatesAppTokenForUserWithExpirationTimeUsingTheAuthAppCLI(
		string $user,
		string $expirationTime
	): void {
		$user = $this->featureContext->getActualUserName($user);
		$command = "auth-app create --user-name=$user --expiration=$expirationTime";
		$body = [
		  "command" => $command
		];
		$this->featureContext->setResponse(CliHelper::runCommand($body));
	}

	/**
	 * @Given user :user has created app token with expiration time :expirationTime using the auth-app CLI
	 *
	 * @param string $user
	 * @param string $expirationTime
	 *
	 * @return void
	 */
	public function userHasCreatedAppTokenWithExpirationTimeUsingTheAuthAppCLI(
		string $user,
		string $expirationTime
	): void {
		$user = $this->featureContext->getActualUserName($user);
		$command = "auth-app create --user-name=$user --expiration=$expirationTime";
		$body = [
		  "command" => $command
		];

		$response = CliHelper::runCommand($body);
		$this->featureContext->theHTTPStatusCodeShouldBe(200, '', $response);
		$jsonResponse = $this->featureContext->getJsonDecodedResponse($response);
		Assert::assertSame("OK", $jsonResponse["status"]);
		Assert::assertSame(
			0,
			$jsonResponse["exitCode"],
			"Expected exit code to be 0, but got " . $jsonResponse["exitCode"]
		);
	}

	/**
	 * @When the administrator removes all the file versions using the CLI
	 *
	 * @return void
	 */
	public function theAdministratorRemovesAllVersionsOfResources() {
		$path = $this->featureContext->getStorageUsersRoot();
		$command = "revisions purge -p $path --dry-run=false";
		$body = [
		  "command" => $command
		];
		$this->featureContext->setResponse(CliHelper::runCommand($body));
	}

	/**
	 * @When the administrator removes the versions of file :file of user :user from space :space using the CLI
	 *
	 * @param string $file
	 * @param string $user
	 * @param string $space
	 *
	 * @return void
	 */
	public function theAdministratorRemovesTheVersionsOfFileUsingFileId($file, $user, $space) {
		$path = $this->featureContext->getStorageUsersRoot();
		$fileId = $this->spacesContext->getFileId($user, $space, $file);
		$command = "revisions purge -p $path -r $fileId --dry-run=false";
		$body = [
		  "command" => $command
		];
		$this->featureContext->setResponse(CliHelper::runCommand($body));
	}

	/**
	 * @When /^the administrator reindexes all spaces using the CLI$/
	 *
	 * @return void
	 */
	public function theAdministratorReindexesAllSpacesUsingTheCli(): void {
		$command = "search index --all-spaces";
		$body = [
		  "command" => $command
		];
		$this->featureContext->setResponse(CliHelper::runCommand($body));
	}

	/**
	 * @When /^the administrator reindexes a space "([^"]*)" using the CLI$/
	 *
	 * @param string $spaceName
	 *
	 * @return void
	 */
	public function theAdministratorReindexesASpaceUsingTheCli(string $spaceName): void {
		$spaceId = $this->spacesContext->getSpaceIdByName($this->featureContext->getAdminUsername(), $spaceName);
		$command = "search index --space $spaceId";
		$body = [
		  "command" => $command
		];
		$this->featureContext->setResponse(CliHelper::runCommand($body));
	}

	/**
	 * @When the administrator removes the file versions of space :space using the CLI
	 *
	 * @param string $space
	 *
	 * @return void
	 */
	public function theAdministratorRemovesTheVersionsOfFilesInSpaceUsingSpaceId(string $space): void {
		$path = $this->featureContext->getStorageUsersRoot();
		$adminUsername = $this->featureContext->getAdminUsername();
		$spaceId = $this->spacesContext->getSpaceIdByName($adminUsername, $space);
		$command = "revisions purge -p $path -r $spaceId --dry-run=false";
		$body = [
		  "command" => $command
		];
		$this->featureContext->setResponse(CliHelper::runCommand($body));
	}

	/**
	 * @Then the command should be successful
	 *
	 * @return void
	 */
	public function theCommandShouldBeSuccessful(): void {
		$response = $this->featureContext->getResponse();
		$this->featureContext->theHTTPStatusCodeShouldBe(200, '', $response);

		$jsonResponse = $this->featureContext->getJsonDecodedResponse($response);

		Assert::assertSame("OK", $jsonResponse["status"]);
		Assert::assertSame(
			0,
			$jsonResponse["exitCode"],
			"Expected exit code to be 0, but got " . $jsonResponse["exitCode"]
		);
	}

	/**
	 * @Then /^the command output (should|should not) contain "([^"]*)"$/
	 *
	 * @param string $shouldOrNot
	 * @param string $output
	 *
	 * @return void
	 */
	public function theCommandOutputShouldContain(string $shouldOrNot, string $output): void {
		$response = $this->featureContext->getResponse();
		$jsonResponse = $this->featureContext->getJsonDecodedResponse($response);
		$output = $this->featureContext->substituteInLineCodes($output);

		if ($shouldOrNot === "should") {
			Assert::assertStringContainsString($output, $jsonResponse["message"]);
		} else {
			Assert::assertStringNotContainsString($output, $jsonResponse["message"]);
		}
	}

	/**
	 * @When the administrator lists all the upload sessions
	 * @When the administrator lists all the upload sessions with flag :flag
	 *
	 * @param string|null $flag
	 *
	 * @return void
	 */
	public function theAdministratorListsAllTheUploadSessions(?string $flag = null): void {
		if ($flag) {
			$flag = "--$flag";
		}
		$command = "storage-users uploads sessions --json $flag";
		$body = [
		  "command" => $command
		];
		$this->featureContext->setResponse(CliHelper::runCommand($body));
	}

	/**
	 * @When the administrator cleans upload sessions with the following flags:
	 *
	 * @param TableNode $table
	 *
	 * @return void
	 */
	public function theAdministratorCleansUploadSessionsWithTheFollowingFlags(TableNode $table): void {
		$flag = "";
		foreach ($table->getRows() as $row) {
			$flag .= "--$row[0] ";
		}
		$flagString = trim($flag);
		$command = "storage-users uploads sessions $flagString --clean --json";
		$body = [
		  "command" => $command
		];
		$this->featureContext->setResponse(CliHelper::runCommand($body));
	}

	/**
	 * @When the administrator restarts the upload sessions that are in postprocessing
	 *
	 * @return void
	 */
	public function theAdministratorRestartsTheUploadSessionsThatAreInPostprocessing(): void {
		$command = "storage-users uploads sessions --processing --restart --json";
		$body = [
		  "command" => $command
		];
		$this->featureContext->setResponse(CliHelper::runCommand($body));
	}

	/**
	 * @When the administrator restarts the upload sessions of file :file
	 *
	 * @param string $file
	 *
	 * @return void
	 * @throws JsonException
	 */
	public function theAdministratorRestartsUploadSessionsOfFile(string $file): void {
		$response = CliHelper::runCommand(["command" => "storage-users uploads sessions --json"]);
		$this->featureContext->theHTTPStatusCodeShouldBe(200, '', $response);
		$responseArray = $this->getJSONDecodedCliMessage($response);

		foreach ($responseArray as $item) {
			if ($item->filename === $file) {
				$uploadId = $item->id;
			}
		}

		$command = "storage-users uploads sessions --id=$uploadId --restart --json";
		$body = [
		  "command" => $command
		];
		$this->featureContext->setResponse(CliHelper::runCommand($body));
	}

	/**
	 * @Then /^the CLI response (should|should not) contain these entries:$/
	 *
	 * @param string $shouldOrNot
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws JsonException
	 */
	public function theCLIResponseShouldContainTheseEntries(string $shouldOrNot, TableNode $table): void {
		$expectedFiles = $table->getColumn(0);
		$responseArray = $this->getJSONDecodedCliMessage($this->featureContext->getResponse());

		$resourceNames = [];
		foreach ($responseArray as $item) {
			if (isset($item->filename)) {
				$resourceNames[] = $item->filename;
			}
		}

		if ($shouldOrNot === "should not") {
			foreach ($expectedFiles as $expectedFile) {
				Assert::assertNotTrue(
					\in_array($expectedFile, $resourceNames),
					"The resource '$expectedFile' was found in the response."
				);
			}
		} else {
			foreach ($expectedFiles as $expectedFile) {
				Assert::assertTrue(
					\in_array($expectedFile, $resourceNames),
					"The resource '$expectedFile' was not found in the response."
				);
			}
		}
	}

	/**
	 * @param ResponseInterface $response
	 *
	 * @return array
	 * @throws JsonException
	 */
	public function getJSONDecodedCliMessage(ResponseInterface $response): array {
		$responseBody = $this->featureContext->getJsonDecodedResponse($response);

		// $responseBody["message"] contains a message info with the array of output json of the upload sessions command
		// Example Output: "INFO memory is not limited, skipping package=github.com/KimMachineGun/automemlimit/memlimit [{<output-json>}]"
		// So, only extracting the array of output json from the message
		\preg_match('/(\[.*\])/', $responseBody["message"], $matches);
		return \json_decode($matches[1], null, 512, JSON_THROW_ON_ERROR);
	}

	/**
	 * @AfterScenario @cli-uploads-sessions
	 *
	 * @return void
	 */
	public function cleanUploadsSessions(): void {
		$command = "storage-users uploads sessions --clean";
		$body = [
		  "command" => $command
		];
		$response = CliHelper::runCommand($body);
		Assert::assertEquals("200", $response->getStatusCode(), "Failed to clean upload sessions");
	}

	/**
	 * @When the administrator creates the folder :folder for user :user on the POSIX filesystem
	 *
	 * @param string $folder
	 * @param string $user
	 *
	 * @return void
	 */
	public function theAdministratorCreatesFolder(string $folder, string $user): void {
		$userUuid = $this->featureContext->getAttributeOfCreatedUser($user, 'id');
		$storagePath = $this->getUsersStoragePath();
		$fullPath = "$storagePath/$userUuid/$folder";

		$body = [
		  "command" => "mkdir -p $fullPath",
		  "raw" => true
		];
		$this->featureContext->setResponse(CliHelper::runCommand($body));
		$this->waitForPath($fullPath);
		sleep(1);
	}

	/**
	 * @When the administrator lists the content of the POSIX storage folder of user :user
	 *
	 * @param string $user
	 *
	 * @return void
	 */
	public function theAdministratorCheckUsersFolder(string $user): void {
		$userUuid = $this->featureContext->getAttributeOfCreatedUser($user, 'id');
		$storagePath = $this->getUsersStoragePath();
		$body = [
		  "command" => "ls -la $storagePath/$userUuid",
		  "raw" => true
		];
		$this->featureContext->setResponse(CliHelper::runCommand($body));
	}

	/**
	 * @When the administrator creates the file :file with content :content for user :user on the POSIX filesystem
	 *
	 * @param string $file
	 * @param string $content
	 * @param string $user
	 *
	 * @return void
	 */
	public function theAdministratorCreatesFile(string $file, string $content, string $user): void {
		$userUuid = $this->featureContext->getAttributeOfCreatedUser($user, 'id');
		$storagePath = $this->getUsersStoragePath();
		$fullPath = "$storagePath/$userUuid/$file";
		$safeContent = escapeshellarg($content);
		$body = [
		  "command" => "echo -n $safeContent > $fullPath",
		  "raw" => true
		];
		$this->featureContext->setResponse(CliHelper::runCommand($body));
		$this->waitForPath($fullPath);
		sleep(1);
	}

	/**
	 * @When the administrator has created the file :file with content :content for user :user on the POSIX filesystem
	 *
	 * @param string $file
	 * @param string $content
	 * @param string $user
	 *
	 * @return void
	 */
	public function theAdministratorHasCreatedFile(string $file, string $content, string $user): void {
		$this->theAdministratorCreatesFile($file, $content, $user);
		$this->theCommandShouldBeSuccessful();
	}

	/**
	 * @When the administrator creates the file :file with size :size for user :user on the POSIX filesystem
	 *
	 * @param string $file
	 * @param string $size Example: "1gb", "5gb"
	 * @param string $user
	 *
	 * @return void
	 */
	public function theAdministratorCreatesLargeFileWithSize(string $file, string $size, string $user): void {
		$userUuid = $this->featureContext->getAttributeOfCreatedUser($user, 'id');
		$storagePath = $this->getUsersStoragePath();

		$size = strtolower($size);
		if (!preg_match('/^(\d+)gb$/', $size, $matches)) {
			throw new \InvalidArgumentException("Invalid size format: $size. Use formats like 1gb, 5gb.");
		}

		$count = (int)$matches[1] * 1024; // 1GB = 1024M
		$bs = '1M';
		$filePath = "$storagePath/$userUuid/$file";

		$body = [
			"command" => "dd if=/dev/zero of=$filePath bs=$bs count=$count",
			"raw" => true
		];
		$this->featureContext->setResponse(CliHelper::runCommand($body));
		$this->waitForPath($filePath, $size, 15);
		sleep(7);
	}

	/**
	 * @When the administrator creates :count files sequentially in the directory :dir for user :user on the POSIX filesystem
	 *
	 * @param int $count
	 * @param string $dir
	 * @param string $user
	 *
	 * @return void
	 */
	public function theAdministratorCreatesFilesSequentially(int $count, string $dir, string $user): void {
		$userUuid = $this->featureContext->getAttributeOfCreatedUser($user, 'id');
		$storagePath = $this->getUsersStoragePath() . "/$userUuid/$dir";
		$cmd = '';
		for ($i = 1; $i <= $count; $i++) {
			$cmd .= "echo -n \"file $i content\" > $storagePath/file_$i.txt; ";
		}
		$body = [
			"command" => $cmd,
			"raw" => true
		];
		$this->featureContext->setResponse(CliHelper::runCommand($body));
		$this->waitForPath("$storagePath/file_$count.txt");
		sleep(3);
	}

	/**
	 * @When the administrator creates :count files in parallel in the directory :dir for user :user on the POSIX filesystem
	 *
	 * @param int $count
	 * @param string $dir
	 * @param string $user
	 *
	 * @return void
	 */
	public function theAdministratorCreatesFilesInParallel(int $count, string $dir, string $user): void {
		$userUuid = $this->featureContext->getAttributeOfCreatedUser($user, 'id');
		$storagePath = $this->getUsersStoragePath() . "/$userUuid/$dir";
		$cmd = "mkdir -p $storagePath; ";
		for ($i = 1; $i <= $count; $i++) {
			$cmd .= "echo -n \"parallel file $i content\" > $storagePath/parallel_$i.txt & ";
		}
		$cmd .= "wait";
		$body = [
			"command" => $cmd,
			"raw" => true
		];
		$this->featureContext->setResponse(CliHelper::runCommand($body));
		$this->waitForPath("$storagePath/parallel_$count.txt");
		sleep(1);
	}

	/**
	 * @When the administrator puts the content :content into the file :file in the POSIX storage folder of user :user
	 *
	 * @param string $content
	 * @param string $file
	 * @param string $user
	 *
	 * @return void
	 */
	public function theAdministratorChangesFileContent(string $content, string $file, string $user): void {
		$userUuid = $this->featureContext->getAttributeOfCreatedUser($user, 'id');
		$storagePath = $this->getUsersStoragePath();
		$safeContent = escapeshellarg($content);
		$body = [
			"command" => "echo -n $safeContent >> $storagePath/$userUuid/$file",
			"raw" => true
		  ];
		sleep(1);
		$this->featureContext->setResponse(CliHelper::runCommand($body));
		sleep(1);
	}

	/**
	 * @When the administrator reads the content of the file :file in the POSIX storage folder of user :user
	 *
	 * @param string $user
	 * @param string $file
	 *
	 * @return void
	 */
	public function theAdministratorReadsTheFileContent(string $user, string $file): void {
		// this downloads the file using WebDAV and by that checks if it's still in
		// postprocessing. So its effectively a check for finished postprocessing
		$this->featureContext->userDownloadsFileUsingTheAPI($user, $file);

		$userUuid = $this->featureContext->getAttributeOfCreatedUser($user, 'id');
		$storagePath = $this->getUsersStoragePath();
		$body = [
		  "command" => "cat $storagePath/$userUuid/$file",
		  "raw" => true
		];
		$this->featureContext->setResponse(CliHelper::runCommand($body));
	}

	/**
	 * @When the administrator copies the file :file to the folder :folder for user :user on the POSIX filesystem
	 *
	 * @param string $user
	 * @param string $file
	 * @param string $folder
	 *
	 * @return void
	 */
	public function theAdministratorCopiesFileToFolder(string $user, string $file, string $folder): void {
		$userUuid = $this->featureContext->getAttributeOfCreatedUser($user, 'id');
		$storagePath = $this->getUsersStoragePath();

		$source = "$storagePath/$userUuid/$file";
		$destination = "$storagePath/$userUuid/$folder";

		$body = [
		  "command" => "cp $source $destination",
		  "raw" => true
		];
		$this->featureContext->setResponse(CliHelper::runCommand($body));
		sleep(1);
	}

	/**
	 * @When the administrator renames the file :file to :newName for user :user on the POSIX filesystem
	 *
	 * @param string $user
	 * @param string $file
	 * @param string $newName
	 *
	 * @return void
	 */
	public function theAdministratorRenamesFile(string $user, string $file, string $newName): void {
		$userUuid = $this->featureContext->getAttributeOfCreatedUser($user, 'id');
		$storagePath = $this->getUsersStoragePath();

		$source = "$storagePath/$userUuid/$file";
		$destination = "$storagePath/$userUuid/$newName";

		$body = [
		  "command" => "mv $source $destination",
		  "raw" => true
		];
		$this->featureContext->setResponse(CliHelper::runCommand($body));
		sleep(1);
	}

	/**
	 * @When the administrator moves the file :file to the folder :folder for user :user on the POSIX filesystem
	 *
	 * @param string $user
	 * @param string $file
	 * @param string $folder
	 *
	 * @return void
	 */
	public function theAdministratorMovesFileToFolder(string $user, string $file, string $folder): void {
		$userUuid = $this->featureContext->getAttributeOfCreatedUser($user, 'id');
		$storagePath = $this->getUsersStoragePath();

		$source = "$storagePath/$userUuid/$file";
		$destination = "$storagePath/$userUuid/$folder";

		$body = [
		  "command" => "mv $source $destination",
		  "raw" => true
		];
		$this->featureContext->setResponse(CliHelper::runCommand($body));
		sleep(1);
	}

	/**
	 * @When the administrator deletes the file :file for user :user on the POSIX filesystem
	 *
	 * @param string $file
	 * @param string $user
	 *
	 * @return void
	 */
	public function theAdministratorDeletesFile(string $file, string $user): void {
		$userUuid = $this->featureContext->getAttributeOfCreatedUser($user, 'id');
		$storagePath = $this->getUsersStoragePath();

		$body = [
		  "command" => "rm $storagePath/$userUuid/$file",
		  "raw" => true
		];
		$this->featureContext->setResponse(CliHelper::runCommand($body));
		sleep(1);
	}

	/**
	 * @When the administrator deletes the folder :folder for user :user on the POSIX filesystem
	 *
	 * @param string $folder
	 * @param string $user
	 *
	 * @return void
	 */
	public function theAdministratorDeletesFolder(string $folder, string $user): void {
		$userUuid = $this->featureContext->getAttributeOfCreatedUser($user, 'id');
		$storagePath = $this->getUsersStoragePath();

		$body = [
		  "command" => "rm -r $storagePath/$userUuid/$folder",
		  "raw" => true
		];
		$this->featureContext->setResponse(CliHelper::runCommand($body));
		sleep(1);
	}

	/**
	 * @When the administrator copies the file :file to the space :space for user :user on the POSIX filesystem
	 *
	 * @param string $user
	 * @param string $file
	 * @param string $space
	 *
	 * @return void
	 */
	public function theAdministratorCopiesFileToSpace(string $user, string $file, string $space): void {
		$userUuid = $this->featureContext->getAttributeOfCreatedUser($user, 'id');
		$usersStoragePath = $this->getUsersStoragePath();
		$projectsStoragePath = $this->getProjectsStoragePath();
		$spaceId = $this->spacesContext->getSpaceIdByName($this->featureContext->getAdminUsername(), $space);
		$spaceId = explode('$', $spaceId)[1];

		$source = "$usersStoragePath/$userUuid/$file";
		$destination = "$projectsStoragePath/$spaceId";

		$body = [
		  "command" => "cp $source $destination",
		  "raw" => true
		];
		$this->featureContext->setResponse(CliHelper::runCommand($body));
		sleep(1);
	}

	/**
	 * @When the administrator deletes the project space :space on the POSIX filesystem
	 *
	 * @param string $space
	 *
	 * @return void
	 */
	public function theAdministratorDeletesSpace(string $space): void {
		$projectsStoragePath = $this->getProjectsStoragePath();
		$spaceId = $this->spacesContext->getSpaceIdByName($this->featureContext->getAdminUsername(), $space);
		$spaceId = explode('$', $spaceId)[1];

		$body = [
		  "command" => "rm -r $projectsStoragePath/$spaceId",
		  "raw" => true
		];
		$this->featureContext->setResponse(CliHelper::runCommand($body));
		sleep(1);
	}

	/**
	 * @When the administrator checks the attribute :attribute of file :file for user :user on the POSIX filesystem
	 *
	 * @param string $attribute
	 * @param string $file
	 * @param string $user
	 *
	 * @return void
	 */
	public function theAdminChecksTheAttributeOfFileForUser(string $attribute, string $file, string $user): void {
		// this downloads the file using WebDAV and by that checks if it's still in
		// postprocessing. So its effectively a check for finished postprocessing
		$this->featureContext->userDownloadsFileUsingTheAPI($user, $file);

		$userUuid = $this->featureContext->getAttributeOfCreatedUser($user, 'id');
		$storagePath = $this->getUsersStoragePath();
		$body = [
			"command" => "xattr -p -slz " . escapeshellarg($attribute) . " $storagePath/$userUuid/$file",
			"raw" => true
		];
		$this->featureContext->setResponse(CliHelper::runCommand($body));
	}
}
