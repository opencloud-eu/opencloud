<?php declare(strict_types=1);
/**
 * @author Joas Schilling <coding@schilljs.com>
 * @author Sergio Bertolin <sbertolin@owncloud.com>
 * @author Phillip Davis <phil@jankaritech.com>
 * @copyright Copyright (c) 2018, ownCloud GmbH
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

use Behat\Gherkin\Node\TableNode;
use Psr\Http\Message\ResponseInterface;
use PHPUnit\Framework\Assert;
use PHPUnit\Framework\ExpectationFailedException;
use TestHelpers\OcsApiHelper;
use TestHelpers\SharingHelper;
use TestHelpers\HttpRequestHelper;
use TestHelpers\TranslationHelper;
use TestHelpers\WebDavHelper;
use GuzzleHttp\Exception\GuzzleException;

/**
 * Sharing trait
 */
trait Sharing {
	private int $sharingApiVersion = 1;

	/**
	 * @var array
	 */
	private array $createdUserGroupShares = [];

	private ?float $localLastShareTime = null;

	/**
	 * Defines the fields that can be provided in a share request.
	 */
	private array $shareFields = [
		'path', 'name', 'publicUpload', 'password', 'expireDate',
		'expireDateAsString', 'permissions', 'shareWith', 'shareType'
	];

	/**
	 * Defines the fields that are known and can be tested in a share response.
	 * Note that OpenCloud also provides file_parent in responses.
	 * file_parent is not provided by opencloud/reva.
	 * There are no known clients that use file_parent.
	 * The acceptance tests do not test for file_parent.
	 *
	 * @var array fields that are possible in a share response
	 */
	private array $shareResponseFields = [
		'id', 'share_type', 'uid_owner', 'displayname_owner', 'stime', 'parent',
		'expiration', 'token', 'uid_file_owner', 'displayname_file_owner', 'path',
		'item_type', 'mimetype', 'storage_id', 'storage', 'item_source',
		'file_source', 'file_target', 'name', 'url', 'mail_send',
		'attributes', 'permissions', 'share_with', 'share_with_displayname', 'share_with_additional_info'
	];

	/**
	 * @var array
	 */
	private array $createdPublicShares = [];

	/**
	 * @var array
	 */
	private array $shareNgCreatedLinkShares = [];

	/**
	 * @var array
	 */
	private array $shareNgCreatedUserGroupShares = [];

	/**
	 * @return string
	 */
	public function getLastCreatedPublicShareToken(): string {
		return (string) $this->getLastCreatedPublicShare()->token;
	}

	/**
	 * @return SimpleXMLElement|null
	 */
	public function getLastCreatedPublicShare(): ?SimpleXMLElement {
		return \end($this->createdPublicShares);
	}

	/**
	 * @param SimpleXMLElement $shareData
	 *
	 * @return void
	 */
	public function addToCreatedPublicShares(SimpleXMLElement $shareData): void {
		$this->createdPublicShares[] = $shareData;
	}

	/**
	 * @param ResponseInterface $response
	 *
	 * @return void
	 */
	public function shareNgAddToCreatedLinkShares(ResponseInterface $response): void {
		$this->shareNgCreatedLinkShares[] = $response;
	}

	/**
	 * @return ResponseInterface|null
	 */
	public function shareNgGetLastCreatedLinkShare(): ?ResponseInterface {
		return \end($this->shareNgCreatedLinkShares);
	}

	/**
	 * @param ResponseInterface $response
	 *
	 * @return void
	 */
	public function shareNgAddToCreatedUserGroupShares(ResponseInterface $response): void {
		$this->shareNgCreatedUserGroupShares[] = $response;
	}

	/**
	 * @return ResponseInterface|null
	 */
	public function shareNgGetLastCreatedUserGroupShare(): ?ResponseInterface {
		return \end($this->shareNgCreatedUserGroupShares);
	}

	/**
	 * @param string $sharer
	 * @param SimpleXMLElement $shareData
	 *
	 * @return void
	 */
	public function addToCreatedUserGroupShares(string $sharer, SimpleXMLElement $shareData): void {
		$this->createdUserGroupShares["$sharer"] = $shareData;
	}

	/**
	 * @return array
	 */
	public function getCreatedUserGroupShares(): array {
		return $this->createdUserGroupShares;
	}

	/**
	 * @return SimpleXMLElement
	 */
	public function getLastCreatedUserGroupShare(): SimpleXMLElement {
		return \end($this->createdUserGroupShares);
	}

	/**
	 * @return void
	 */
	public function emptyCreatedPublicShares(): void {
		$this->createdPublicShares = [];
	}

	/**
	 * @return void
	 */
	public function emptyCreatedUserGroupShares(): void {
		$this->createdUserGroupShares = [];
	}

	/**
	 * @return float|null
	 */
	public function getLocalLastShareTime(): ?float {
		return $this->localLastShareTime;
	}

	/**
	 * @param string|null $postfix string to append to the end of the path
	 *
	 * @return string
	 */
	public function getSharesEndpointPath(?string $postfix = ''): string {
		return "/apps/files_sharing/api/v$this->sharingApiVersion/shares$postfix";
	}

	/**
	 * @return string
	 */
	public function shareNgGetLastCreatedLinkShareID(): string {
		$lastResponse = $this->shareNgGetLastCreatedLinkShare();
		if (!isset($this->getJsonDecodedResponse($lastResponse)['id'])) {
			throw new Error('Response did not contain share id for the created public link');
		}
		return $this->getJsonDecodedResponse($lastResponse)['id'];
	}

	/**
	 * @return string
	 */
	public function shareNgGetLastCreatedLinkShareToken(): string {
		$lastResponse = $this->shareNgGetLastCreatedLinkShare();
		if (!isset($this->getJsonDecodedResponse($lastResponse)['link']['webUrl'])) {
			throw new Error(
				'Response did not contain share id '
				. $this->getJsonDecodedResponse($lastResponse)['link']['webUrl']
				. ' for the created public link'
			);
		}
		$last_created_link_webURL =  $this->getJsonDecodedResponse($lastResponse)['link']['webUrl'];
		return substr(strrchr($last_created_link_webURL, "/"), 1);
	}

	/**
	 * @return string
	 */
	public function shareNgGetLastCreatedUserGroupShareID(): string {
		$lastResponse = $this->shareNgGetLastCreatedUserGroupShare();
		if (!isset($this->getJsonDecodedResponse($lastResponse)['value'][0]['id'])) {
			throw new Error('Response did not contain share id for the last created share.');
		}
		return $this->getJsonDecodedResponse($lastResponse)['value'][0]['id'];
	}

	/**
	 * Split given permissions string each separated with "," into an array of strings
	 *
	 * @param string $str
	 *
	 * @return string[]
	 */
	private function splitPermissionsString(string $str): array {
		$str = \trim($str);
		$permissions = \array_map('trim', \explode(',', $str));

		/* We use 'all', 'uploadwriteonly' and 'change' in feature files
		for readability. Parse into appropriate permissions and return them
		without any duplications.*/
		if (\in_array('all', $permissions, true)) {
			$permissions = \array_keys(SharingHelper::PERMISSION_TYPES);
		}
		if (\in_array('uploadwriteonly', $permissions, true)) {
			// remove 'uploadwriteonly' from $permissions
			$permissions = \array_diff($permissions, ['uploadwriteonly']);
			$permissions = \array_merge($permissions, ['create']);
		}
		if (\in_array('change', $permissions, true)) {
			// remove 'change' from $permissions
			$permissions = \array_diff($permissions, ['change']);
			$permissions = \array_merge(
				$permissions,
				['create', 'delete', 'read', 'update']
			);
		}

		return \array_unique($permissions);
	}

	/**
	 *
	 * @return int
	 *
	 * @throws Exception
	 */
	public function getServerShareTimeFromLastResponse(): int {
		$stime = HttpRequestHelper::getResponseXml($this->response, __METHOD__)->xpath("//stime");
		if ($stime) {
			return (int) $stime[0];
		}
		throw new Exception("Last share time (i.e. 'stime') could not be found in the response.");
	}

	/**
	 * @return void
	 */
	private function waitToCreateShare(): void {
		if (($this->localLastShareTime !== null)
			&& ((\microtime(true) - $this->localLastShareTime) < 1)
		) {
			// prevent creating two shares with the same "stime" which is
			// based on seconds, this affects share merging order and could
			// affect expected test result order
			\sleep(1);
		}
	}

	/**
	 * @param string $user
	 * @param TableNode|null $body
	 *    TableNode $body should not have any heading and can have the following rows   |
	 *       | path               | The folder or file path to be shared                |
	 *       | name               | A (human-readable) name for the share,              |
	 *       |                    | which can be up to 64 characters in length.         |
	 *       | publicUpload       | Whether to allow public upload to a public          |
	 *       |                    | shared folder. Write true for allowing.             |
	 *       | password           | The password to protect the public link share with. |
	 *       | expireDate         | An expire date for public link shares.              |
	 *       |                    | This argument takes a date string in any format     |
	 *       |                    | that can be passed to strtotime(), for example:     |
	 *       |                    | 'YYYY-MM-DD' or '+ x days'. It will be converted to |
	 *       |                    | 'YYYY-MM-DD' format before sending                  |
	 *       | expireDateAsString | An expire date string for public link shares.       |
	 *       |                    | Whatever string is provided will be sent as the     |
	 *       |                    | expire date. For example, use this to test sending  |
	 *       |                    | invalid date strings.                               |
	 *       | permissions        | The permissions to set on the share.                |
	 *       |                    |     1 = read; 2 = update; 4 = create;               |
	 *       |                    |     8 = delete; 16 = share; 31 = all                |
	 *       |                    |     15 = change; 0 = invite                         |
	 *       |                    |     4 = uploadwriteonly                             |
	 *       |                    |     (default: 31, for public shares: 1)             |
	 *       |                    |     Pass either the (total) number,                 |
	 *       |                    |     or the keyword,                                 |
	 *       |                    |     or a comma separated list of keywords           |
	 *       | shareWith          | The user or group id with which the file should     |
	 *       |                    | be shared.                                          |
	 *       | shareType          | The type of the share. This can be one of:          |
	 *       |                    |    0 = user, 1 = group, 3 = public_link,            |
	 *       |                    |    6 = federated (cloud share).                     |
	 *       |                    |    Pass either the number or the keyword.           |
	 *
	 * @return ResponseInterface
	 * @throws Exception
	 */
	public function createShareWithSettings(string $user, ?TableNode $body): ResponseInterface {
		$user = $this->getActualUsername($user);
		$this->verifyTableNodeRows(
			$body,
			['path'],
			$this->shareFields
		);
		$bodyRows = $body->getRowsHash();
		$bodyRows['name'] = \array_key_exists('name', $bodyRows) ? $bodyRows['name'] : null;
		$bodyRows['shareWith'] = \array_key_exists('shareWith', $bodyRows) ? $bodyRows['shareWith'] : null;
		$bodyRows['shareWith'] = $this->getActualUsername($bodyRows['shareWith']);
		$bodyRows['publicUpload'] = \array_key_exists(
			'publicUpload',
			$bodyRows
		) ? $bodyRows['publicUpload'] === 'true' : null;
		$bodyRows['password'] = \array_key_exists(
			'password',
			$bodyRows
		) ? $this->getActualPassword($bodyRows['password']) : null;

		if (\array_key_exists('permissions', $bodyRows)) {
			if (\is_numeric($bodyRows['permissions'])) {
				$bodyRows['permissions'] = (int) $bodyRows['permissions'];
			} else {
				$bodyRows['permissions'] = $this->splitPermissionsString($bodyRows['permissions']);
			}
		} else {
			$bodyRows['permissions'] = null;
		}
		if (\array_key_exists('shareType', $bodyRows)) {
			if (\is_numeric($bodyRows['shareType'])) {
				$bodyRows['shareType'] = (int) $bodyRows['shareType'];
			}
		} else {
			$bodyRows['shareType'] = null;
		}

		Assert::assertFalse(
			isset($bodyRows['expireDate'], $bodyRows['expireDateAsString']),
			'expireDate and expireDateAsString cannot be set at the same time.'
		);
		$needToParse = \array_key_exists('expireDate', $bodyRows);
		$expireDate = $bodyRows['expireDate'] ?? $bodyRows['expireDateAsString'] ?? null;
		$bodyRows['expireDate'] = $needToParse ? \date('Y-m-d', \strtotime($expireDate)) : $expireDate;
		return $this->createShare(
			$user,
			$bodyRows['path'],
			$bodyRows['shareType'],
			$bodyRows['shareWith'],
			$bodyRows['publicUpload'],
			$bodyRows['password'],
			$bodyRows['permissions'],
			$bodyRows['name'],
			$bodyRows['expireDate']
		);
	}

	/**
	 * @When /^user "([^"]*)" creates a share using the sharing API with settings$/
	 *
	 * @param string $user
	 * @param TableNode|null $body {@link createShareWithSettings}
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userCreatesAShareWithSettings(string $user, ?TableNode $body): void {
		$user = $this->getActualUsername($user);
		$response = $this->createShareWithSettings(
			$user,
			$body
		);
		$this->setResponse($response);
		$this->pushToLastStatusCodesArrays();
	}

	/**
	 * @param string $user
	 * @param TableNode $body
	 *
	 * @return ResponseInterface
	 */
	public function createPublicLinkShare(string $user, TableNode $body): ResponseInterface {
		$rows = $body->getRows();
		// A public link share is shareType 3
		$rows[] = ['shareType', 'public_link'];
		$newBody = new TableNode($rows);
		return $this->createShareWithSettings($user, $newBody);
	}

	/**
	 * @When /^user "([^"]*)" creates a public link share using the sharing API with settings$/
	 *
	 * @param string $user
	 * @param TableNode $body
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userCreatesAPublicLinkShareWithSettings(string $user, TableNode $body): void {
		$this->setResponse($this->createPublicLinkShare($user, $body));
		$this->pushToLastStatusCodesArrays();
	}

	/**
	 * @param string $user
	 * @param string $path
	 * @param boolean $publicUpload
	 * @param string|null $sharePassword
	 * @param string|int|string[]|int[]|null $permissions
	 * @param string|null $linkName
	 * @param string|null $expireDate
	 *
	 * @return ResponseInterface
	 */
	public function createAPublicShare(
		string $user,
		string $path,
		bool $publicUpload = false,
		string $sharePassword = null,
		$permissions = null,
		?string $linkName = null,
		?string $expireDate = null
	): ResponseInterface {
		return $this->createShare(
			$user,
			$path,
			'public_link',
			null, // shareWith
			$publicUpload,
			$sharePassword,
			$permissions,
			$linkName,
			$expireDate
		);
	}

	/**
	 * @param string $user
	 * @param string $path
	 * @param string $expiryDate in a valid date format, e.g. "+30 days"
	 *
	 * @return void
	 */
	public function createPublicLinkShareOfResourceWithExpiry(
		string $user,
		string $path,
		string $expiryDate
	): void {
		$this->createAPublicShare(
			$user,
			$path,
			true,
			null,
			null,
			null,
			$expiryDate
		);
	}

	/**
	 * Gets the last user share id using the sharer's name
	 *
	 * @param string|null $user
	 *
	 * @return string
	 * @throws Exception
	 */
	public function getLastCreatedUserGroupShareId(?string $user = null): string {
		if ($user === null) {
			$shareId = $this->isUsingSharingNG()
			? $this->shareNgGetLastCreatedUserGroupShareID() : $this->getLastCreatedUserGroupShare()->id;
			return (string) $shareId;
		}
		$createdShares = $this->getCreatedUserGroupShares();
		if (isset($createdShares[$user])) {
			return (string) $createdShares[$user]->id;
		}
		throw new Exception(__METHOD__ . " user '$user' doesn't have a share in the created shares list");
	}

	/**
	 * @param string $user
	 * @param TableNode|null $body
	 * @param string|null $shareOwner
	 * @param bool $updateLastPublicLink
	 *
	 * @return ResponseInterface
	 * @throws Exception
	 */
	public function updateLastShareWithSettings(
		string $user,
		?TableNode $body,
		?string $shareOwner = null,
		?bool $updateLastPublicLink = false
	): ResponseInterface {
		$user = $this->getActualUsername($user);

		if ($updateLastPublicLink) {
			$share_id = ($this->isUsingSharingNG())
			? $this->shareNgGetLastCreatedLinkShareID() : (string) $this->getLastCreatedPublicShare()->id;
		} else {
			if ($shareOwner === null) {
				$share_id = ($this->isUsingSharingNG())
				? $this->shareNgGetLastCreatedUserGroupShareID() : $this->getLastCreatedUserGroupShareId();
			} else {
				$share_id = $this->getLastCreatedUserGroupShareId($shareOwner);
			}
		}

		$this->verifyTableNodeRows(
			$body,
			[],
			$this->shareFields
		);
		$bodyRows = $body->getRowsHash();

		if (\array_key_exists('password', $bodyRows)) {
			$bodyRows['password'] = $this->getActualPassword($bodyRows['password']);
		}
		if (\array_key_exists('permissions', $bodyRows)) {
			if (\is_numeric($bodyRows['permissions'])) {
				$bodyRows['permissions'] = (int) $bodyRows['permissions'];
			} else {
				$bodyRows['permissions'] = $this->splitPermissionsString($bodyRows['permissions']);
				$bodyRows['permissions'] = SharingHelper::getPermissionSum($bodyRows['permissions']);
			}
		}
		return OcsApiHelper::sendRequest(
			$this->getBaseUrl(),
			$user,
			$this->getPasswordForUser($user),
			"PUT",
			$this->getSharesEndpointPath("/$share_id"),
			$this->getStepLineRef(),
			$bodyRows,
			$this->ocsApiVersion
		);
	}

	/**
	 * @When /^user "([^"]*)" updates the last share using the sharing API with$/
	 *
	 * @param string $user
	 * @param TableNode|null $body
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userUpdatesTheLastShareWith(string $user, ?TableNode $body): void {
		$this->setResponse($this->updateLastShareWithSettings($user, $body));
		$this->pushToLastStatusCodesArrays();
	}

	/**
	 * @When /^user "([^"]*)" updates the last public link share using the sharing API with$/
	 *
	 * @param string $user
	 * @param TableNode|null $body
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userUpdatesTheLastPublicLinkShareWith(string $user, ?TableNode $body): void {
		$this->response = $this->updateLastShareWithSettings($user, $body, null, true);
		$this->pushToLastStatusCodesArrays();
	}

	/**
	 * @Given /^user "([^"]*)" has updated the last share with$/
	 *
	 * @param string $user
	 * @param TableNode|null $body
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userHasUpdatedTheLastShareWith(string $user, ?TableNode $body): void {
		$response = $this->updateLastShareWithSettings($user, $body);
		$this->theHTTPStatusCodeShouldBeBetween(200, 299, $response);
	}

	/**
	 * @param string $user
	 * @param string|null $path
	 * @param string|null $shareType
	 * @param string|null $shareWith
	 * @param bool|null $publicUpload
	 * @param string|null $sharePassword
	 * @param string|int|string[]|int[]|null $permissions
	 * @param string|null $linkName
	 * @param string|null $expireDate
	 * @param string|null $space_ref
	 * @param string $sharingApp
	 *
	 * @return ResponseInterface
	 * @throws JsonException
	 * @throws Exception
	 */
	public function createShare(
		string $user,
		?string $path = null,
		?string $shareType = null,
		?string $shareWith = null,
		?bool $publicUpload = null,
		?string $sharePassword = null,
		$permissions = null,
		?string $linkName = null,
		?string $expireDate = null,
		?string $space_ref = null,
		string $sharingApp = 'files_sharing'
	): ResponseInterface {
		$userActual = $this->getActualUsername($user);
		if (\is_string($permissions) && !\is_numeric($permissions)) {
			$permissions = $this->splitPermissionsString($permissions);
		}
		$this->waitToCreateShare();
		$response = SharingHelper::createShare(
			$this->getBaseUrl(),
			$userActual,
			$this->getPasswordForUser($user),
			$path,
			$shareType,
			$this->getStepLineRef(),
			$shareWith,
			$publicUpload,
			$sharePassword,
			$permissions,
			$linkName,
			$expireDate,
			$space_ref,
			$this->ocsApiVersion,
			$this->sharingApiVersion,
			$sharingApp
		);

		// save the created share data
		if (($response->getStatusCode() === 200)
			&& \in_array($this->ocsContext->getOCSResponseStatusCode($response), ['100', '200'])
		) {
			$responseXmlObject = HttpRequestHelper::getResponseXml($response);
			if (isset($responseXmlObject->data)) {
				$shareData = $responseXmlObject->data;
				if ($shareType === 'public_link') {
					$this->addToCreatedPublicShares($shareData);
				} else {
					$sharer = (string) $responseXmlObject->data->uid_owner;
					$this->addToCreatedUserGroupshares($sharer, $shareData);
				}
			}
		}
		$this->localLastShareTime = \microtime(true);
		return $response;
	}

	/**
	 * @param string $field
	 * @param string $value
	 * @param string $contentExpected
	 * @param bool $expectSuccess if true then the caller expects that the field
	 *                            has the expected content
	 *                            emit debugging information if the field is not as expected
	 *
	 * @return bool
	 */
	public function doesFieldValueMatchExpectedContent(
		string $field,
		string $value,
		string $contentExpected,
		bool $expectSuccess = true
	): bool {
		if (($contentExpected === "ANY_VALUE")
			|| (($contentExpected === "A_TOKEN") && (\strlen($value) === 15))
			|| (($contentExpected === "A_NUMBER") && \is_numeric($value))
			|| (($contentExpected === "A_STRING") && $value !== "")
			|| (($contentExpected === "AN_URL") && $this->isAPublicLinkUrl($value))
			|| (($field === 'remote') && (\rtrim($value, "/") === $contentExpected))
			|| ($contentExpected === $value)
		) {
			if (!$expectSuccess) {
				echo $field . " is unexpectedly set with value '" . $value . "'\n";
			}
			return true;
		}
		return false;
	}

	/**
	 * @param string $field
	 * @param string|null $contentExpected
	 * @param bool $expectSuccess if true then the caller expects that the field
	 *                            is in the response with the expected content
	 *                            so emit debugging information if the field is not correct
	 * @param SimpleXMLElement|null $data
	 *
	 * @return bool
	 * @throws Exception
	 */
	public function isFieldInResponse(
		string $field,
		?string $contentExpected,
		bool $expectSuccess = true,
		?SimpleXMLElement $data = null
	): bool {
		if ($data === null) {
			$data = HttpRequestHelper::getResponseXml($this->response, __METHOD__)->data[0];
		}
		Assert::assertIsObject($data, __METHOD__ . " data not found in response XML");

		$dateFieldsArrayToConvert = ['original_date', 'new_date'];
		//do not try to convert empty date
		if ((string) \in_array($field, \array_merge($dateFieldsArrayToConvert)) && !empty($contentExpected)) {
			$timestamp = \strtotime($contentExpected, $this->getServerShareTimeFromLastResponse());
			// strtotime returns false if it failed to parse, just leave it as it is in that condition
			if ($timestamp !== false) {
				$contentExpected
					= \date(
						'Y-m-d',
						$timestamp
					) . " 00:00:00";
			}
		}
		$contentExpected = (string) $contentExpected;

		if (\count($data->element) > 0) {
			$fieldIsSet = false;
			$value = "";
			foreach ($data as $element) {
				if (isset($element->$field)) {
					$fieldIsSet = true;
					$value = (string) $element->$field;
					// convert expiration to Y-m-d format. bug #5424
					if ($field === "expiration") {
						$value = (preg_split("/[\sT]+/", $value))[0];
					}
					if ($this->doesFieldValueMatchExpectedContent(
						$field,
						$value,
						$contentExpected,
						$expectSuccess
					)
					) {
						return true;
					}
				}
			}
		} else {
			$fieldIsSet = isset($data->$field);
			if ($fieldIsSet) {
				$value = (string) $data->$field;
				if ($this->doesFieldValueMatchExpectedContent(
					$field,
					$value,
					$contentExpected,
					$expectSuccess
				)
				) {
					return true;
				}
			}
		}
		if ($expectSuccess) {
			if ($fieldIsSet) {
				echo $field . " has unexpected value '" . $value . "'\n";
			} else {
				echo $field . " is not set in response\n";
			}
		}
		return false;
	}

	/**
	 * @Then no files or folders should be included in the response
	 *
	 * @return void
	 */
	public function checkNoFilesFoldersInResponse(): void {
		$data = HttpRequestHelper::getResponseXml($this->response, __METHOD__)->data[0];
		Assert::assertIsObject($data, __METHOD__ . " data not found in response XML");
		Assert::assertCount(0, $data);
	}

	/**
	 * @Then exactly :count file/files or folder/folders should be included in the response
	 *
	 * @param string $count
	 *
	 * @return void
	 */
	public function checkCountFilesFoldersInResponse(string $count): void {
		$count = (int) $count;
		$data = HttpRequestHelper::getResponseXml($this->response, __METHOD__)->data[0];
		Assert::assertIsObject($data, __METHOD__ . " data not found in response XML");
		Assert::assertCount($count, $data, __METHOD__ . " the response does not contain $count entries");
	}

	/**
	 * @Then /^(?:file|folder|entry) "([^"]*)" should be included in the response$/
	 *
	 * @param string $filename
	 *
	 * @return void
	 * @throws Exception
	 */
	public function checkSharedFileInResponse(string $filename): void {
		$filename = "/" . \ltrim($filename, '/');
		Assert::assertTrue(
			$this->isFieldInResponse('file_target', "$filename"),
			"'file_target' value '$filename' was not found in response"
		);
	}

	/**
	 * @Then /^(?:file|folder|entry) "([^"]*)" should not be included in the response$/
	 *
	 * @param string $filename
	 *
	 * @return void
	 * @throws Exception
	 */
	public function checkSharedFileNotInResponse(string $filename): void {
		$filename = "/" . \ltrim($filename, '/');
		Assert::assertFalse(
			$this->isFieldInResponse('file_target', "$filename", false),
			"'file_target' value '$filename' was unexpectedly found in response"
		);
	}

	/**
	 * @Then /^(?:file|folder|entry) "([^"]*)" should be included as path in the response$/
	 *
	 * @param string $filename
	 *
	 * @return void
	 * @throws Exception
	 */
	public function checkSharedFileAsPathInResponse(string $filename): void {
		$filename = "/" . \ltrim($filename, '/');
		Assert::assertTrue(
			$this->isFieldInResponse('path', "$filename"),
			"'path' value '$filename' was not found in response"
		);
	}

	/**
	 * @Then /^(?:file|folder|entry) "([^"]*)" should not be included as path in the response$/
	 *
	 * @param string $filename
	 *
	 * @return void
	 * @throws Exception
	 */
	public function checkSharedFileAsPathNotInResponse(string $filename): void {
		$filename = "/" . \ltrim($filename, '/');
		Assert::assertFalse(
			$this->isFieldInResponse('path', "$filename", false),
			"'path' value '$filename' was unexpectedly found in response"
		);
	}

	/**
	 * @Then /^(user|group) "([^"]*)" should be included in the response$/
	 *
	 * @param string $type
	 * @param string $user
	 *
	 * @return void
	 * @throws Exception
	 */
	public function checkSharedUserOrGroupInResponse(string $type, string $user): void {
		if ($type === 'user') {
			$user = $this->getActualUsername($user);
		}
		Assert::assertTrue(
			$this->isFieldInResponse('share_with', "$user"),
			"'share_with' value '$user' was not found in response"
		);
	}

	/**
	 * @Then /^user "([^"]*)" should not be included in the response$/
	 * @Then /^group "([^"]*)" should not be included in the response$/
	 *
	 * @param string $userOrGroup
	 *
	 * @return void
	 * @throws Exception
	 */
	public function checkSharedUserOrGroupNotInResponse(string $userOrGroup): void {
		Assert::assertFalse(
			$this->isFieldInResponse('share_with', "$userOrGroup", false),
			"'share_with' value '$userOrGroup' was unexpectedly found in response"
		);
	}

	/**
	 *
	 * @param string $sharer
	 * @param string $filepath
	 * @param string $sharee
	 * @param string|int|string[]|int[] $permissions
	 *
	 * @return ResponseInterface
	 */
	public function createAUserShare(
		string $sharer,
		string $filepath,
		string $sharee,
		$permissions = null
	): ResponseInterface {
		return $this->createShare(
			$sharer,
			$filepath,
			'0',
			$this->getActualUsername($sharee),
			null,
			null,
			$permissions
		);
	}

	/**
	 * @When /^user "([^"]*)" shares (?:file|folder|entry) "([^"]*)" with user "([^"]*)"(?: with permissions (\d+))? using the sharing API$/
	 * @When /^user "([^"]*)" shares (?:file|folder|entry) "([^"]*)" with user "([^"]*)" with permissions "([^"]*)" using the sharing API$/
	 *
	 * @param string $sharer
	 * @param string $filepath
	 * @param string $sharee
	 * @param string|int|string[]|int[] $permissions
	 *
	 * @return void
	 */
	public function userSharesFileWithUserUsingTheSharingApi(
		string $sharer,
		string $filepath,
		string $sharee,
		$permissions = null
	): void {
		$response = $this->createAUserShare(
			$sharer,
			$filepath,
			$this->getActualUsername($sharee),
			$permissions
		);
		$this->setResponse($response);
		$this->pushToLastStatusCodesArrays();
	}

	/**
	 * @When /^user "([^"]*)" shares the following (?:files|folders|entries) with user "([^"]*)"(?: with permissions (\d+))? using the sharing API$/
	 * @When /^user "([^"]*)" shares the following (?:files|folders|entries) with user "([^"]*)" with permissions "([^"]*)" using the sharing API$/
	 *
	 * @param string $sharer
	 * @param string $sharee
	 * @param TableNode $table
	 * @param string|int|string[]|int[] $permissions
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userSharesTheFollowingFilesWithUserUsingTheSharingApi(
		string $sharer,
		string $sharee,
		TableNode $table,
		$permissions = null
	): void {
		$this->verifyTableNodeColumns($table, ["path"]);
		$paths = $table->getHash();

		foreach ($paths as $filepath) {
			$response = $this->createAUserShare(
				$sharer,
				$filepath["path"],
				$this->getActualUsername($sharee),
				$permissions
			);
			$this->setResponse($response);
			$this->pushToLastStatusCodesArrays();
		}
	}

	/**
	 * @When /^the user shares (?:file|folder|entry) "([^"]*)" with user "([^"]*)"(?: with permissions (\d+))? using the sharing API$/
	 * @When /^the user shares (?:file|folder|entry) "([^"]*)" with user "([^"]*)" with permissions "([^"]*)" using the sharing API$/
	 *
	 * @param string $filepath
	 * @param string $user2
	 * @param string|int|string[]|int[] $permissions
	 *
	 * @return void
	 */
	public function theUserSharesFileWithUserUsingTheSharingApi(
		string $filepath,
		string $user2,
		$permissions = null
	) {
		$response = $this->createAUserShare(
			$this->getCurrentUser(),
			$filepath["path"],
			$this->getActualUsername($user2),
			$permissions
		);
		$this->setResponse($response);
		$this->pushToLastStatusCodesArrays();
	}

	/**
	 *
	 * @param string $user
	 * @param string $filepath
	 * @param string $group
	 * @param string|int|string[]|int[] $permissions
	 *
	 * @return ResponseInterface
	 */
	public function createAGroupShare(
		string $user,
		string $filepath,
		string$group,
		$permissions = null
	): ResponseInterface {
		return $this->createShare(
			$user,
			$filepath,
			'1',
			$group,
			null,
			null,
			$permissions
		);
	}

	/**
	 * @When /^user "([^"]*)" shares (?:file|folder|entry) "([^"]*)" with group "([^"]*)" with permissions "([^"]*)" using the sharing API$/
	 * @When /^user "([^"]*)" shares (?:file|folder|entry) "([^"]*)" with group "([^"]*)"(?: with permissions (\d+))? using the sharing API$/
	 *
	 * @param string $user
	 * @param string $filepath
	 * @param string $group
	 * @param string|int|string[]|int[] $permissions
	 *
	 * @return void
	 */
	public function userSharesFileWithGroupUsingTheSharingApi(
		string $user,
		string $filepath,
		string $group,
		$permissions = null
	) {
		$response = $this->createAGroupShare(
			$user,
			$filepath,
			$group,
			$permissions
		);
		$this->setResponse($response);
		$this->pushToLastStatusCodesArrays();
	}

	/**
	 * @When /^user "([^"]*)" shares the following (?:files|folders|entries) with group "([^"]*)"(?: with permissions (\d+))? using the sharing API$/
	 * @When /^user "([^"]*)" shares the following (?:files|folders|entries) with group "([^"]*)" with permissions "([^"]*)" using the sharing API$/
	 *
	 * @param string $user
	 * @param string $group
	 * @param TableNode $table
	 * @param string|int|string[]|int[] $permissions
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userSharesTheFollowingFilesWithGroupUsingTheSharingApi(
		string $user,
		string $group,
		TableNode $table,
		$permissions = null
	) {
		$this->verifyTableNodeColumns($table, ["path"]);
		$paths = $table->getHash();

		foreach ($paths as $filepath) {
			$response = $this->createAGroupShare(
				$user,
				$filepath["path"],
				$group,
				$permissions
			);
			$this->setResponse($response);
			$this->pushToLastStatusCodesArrays();
		}
	}

	/**
	 * @Then /^user "([^"]*)" should not be able to share (?:file|folder|entry) "([^"]*)" with (user|group) "([^"]*)"(?: with permissions (\d+))? using the sharing API$/
	 * @Then /^user "([^"]*)" should not be able to share (?:file|folder|entry) "([^"]*)" with (user|group) "([^"]*)" with permissions "([^"]*)" using the sharing API$/
	 *
	 * @param string $sharer
	 * @param string $filepath
	 * @param string $userOrGroupShareType
	 * @param string $sharee
	 * @param string|int|string[]|int[] $permissions
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userTriesToShareFileUsingTheSharingApi(
		string $sharer,
		string $filepath,
		string $userOrGroupShareType,
		string $sharee,
		$permissions = null
	): void {
		$sharee = $this->getActualUsername($sharee);
		$response = $this->createShare(
			$sharer,
			$filepath,
			$userOrGroupShareType,
			$sharee,
			null,
			null,
			$permissions
		);
		$statusCode = $this->ocsContext->getOCSResponseStatusCode($response);
		Assert::assertTrue(
			($statusCode == 404) || ($statusCode == 403),
			"Sharing should have failed with status code 403 or 404 but got status code $statusCode"
		);
	}

	/**
	 * @Then /^user "([^"]*)" should be able to share (?:file|folder|entry) "([^"]*)" with (user|group) "([^"]*)"(?: with permissions (\d+))? using the sharing API$/
	 * @Then /^user "([^"]*)" should be able to share (?:file|folder|entry) "([^"]*)" with (user|group) "([^"]*)" with permissions "([^"]*)" using the sharing API$/
	 *
	 * @param string $sharer
	 * @param string $filepath
	 * @param string $userOrGroupShareType
	 * @param string $sharee
	 * @param string|int|string[]|int[] $permissions
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userShouldBeAbleToShareUsingTheSharingApi(
		string $sharer,
		string $filepath,
		string $userOrGroupShareType,
		string $sharee,
		$permissions = null
	): void {
		$sharee = $this->getActualUsername($sharee);
		$response = $this->createShare(
			$sharer,
			$filepath,
			$userOrGroupShareType,
			$sharee,
			null,
			null,
			$permissions
		);

		$statusCode = $this->ocsContext->getOCSResponseStatusCode($response);
		Assert::assertTrue(
			($statusCode == 100) || ($statusCode == 200),
			"Sharing should be successful but got ocs status code $statusCode"
		);
	}

	/**
	 * @param string $user the user who will do the delete request
	 * @param string|null $sharer the specific user whose share will be deleted (if specified)
	 * @param bool $deleteLastPublicLink
	 *
	 * @return ResponseInterface
	 */
	public function deleteLastShareUsingSharingApi(
		string $user,
		string $sharer = null,
		bool $deleteLastPublicLink = false
	): ResponseInterface {
		$user = $this->getActualUsername($user);
		if ($deleteLastPublicLink) {
			$shareId = (string) $this->getLastCreatedPublicShare()->id;
		} else {
			if ($sharer === null) {
				$shareId = ($this->isUsingSharingNG())
				? $this->shareNgGetLastCreatedUserGroupShareID() : $this->getLastCreatedUserGroupShareId();
			} else {
				$shareId = ($this->isUsingSharingNG())
				? $this->shareNgGetLastCreatedUserGroupShareID() : $this->getLastCreatedUserGroupShareId($sharer);
			}
		}
		$url = $this->getSharesEndpointPath("/$shareId");
		return $this->ocsContext->sendRequestToOcsEndpoint(
			$user,
			"DELETE",
			$url
		);
	}

	/**
	 * @return ResponseInterface
	 */
	public function deleteLastShareUsingSharingApiByCurrentUser(): ResponseInterface {
		return $this->deleteLastShareUsingSharingApi($this->currentUser);
	}

	/**
	 * @When /^user "([^"]*)" deletes the last share using the sharing API$/
	 *
	 * @param string $user
	 *
	 * @return void
	 */
	public function userDeletesLastShareUsingTheSharingApi(string $user): void {
		$this->setResponse($this->deleteLastShareUsingSharingApi($user));
		$this->pushToLastStatusCodesArrays();
	}

	/**
	 * @When /^user "([^"]*)" deletes the last share of user "([^"]*)" using the sharing API$/
	 * @When /^user "([^"]*)" tries to delete the last share of user "([^"]*)" using the sharing API$/
	 *
	 * @param string $user
	 * @param string $sharer
	 *
	 * @return void
	 */
	public function userDeletesLastShareOfUserUsingTheSharingApi(string $user, string $sharer): void {
		$this->setResponse($this->deleteLastShareUsingSharingApi($user, $sharer));
		$this->pushToLastStatusCodesArrays();
	}

	/**
	 * @Given /^user "([^"]*)" has deleted the last share$/
	 *
	 * @param string $user
	 *
	 * @return void
	 */
	public function userHasDeletedLastShareUsingTheSharingApi(string $user): void {
		$response = $this->deleteLastShareUsingSharingApi($user);
		$this->theHTTPStatusCodeShouldBeBetween(200, 299, $response);
	}

	/**
	 * @param string $user
	 * @param string $shareType    user|group|link
	 * @param string|null $language
	 *
	 * @return ResponseInterface
	 */
	public function getLastShareInfo(string $user, string $shareType, ?string $language = null): ResponseInterface {
		if ($shareType !== "link") {
			$shareId = $this->isUsingSharingNg()
			? $this->shareNgGetLastCreatedUserGroupShareID() : $this->getLastCreatedUserGroupShareId();
		} else {
			$shareId = ($this->isUsingSharingNG())
			? $this->shareNgGetLastCreatedLinkShareID() : (string) $this->getLastCreatedPublicShare()->id;
		}
		if ($shareId === null) {
			throw new Exception(
				__METHOD__ . " last public link share data was not found"
			);
		}
		$language = TranslationHelper::getLanguage($language);
		return $this->getShareData($user, $shareId, $language);
	}

	/**
	 * @When /^user "([^"]*)" gets the info of the last share using the sharing API$/
	 *
	 * @param string $user username that requests the information (might not be the user that has initiated the share)
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userGetsInfoOfLastShareUsingTheSharingApi(string $user): void {
		$response = $this->getLastShareInfo($user, "user");
		$this->setResponse($response);
		$this->pushToLastStatusCodesArrays();
	}

	/**
	 * @When /^user "([^"]*)" gets the info of the last public link share using the sharing API$/
	 *
	 * @param string $user username that requests the information (might not be the user that has initiated the share)
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userGetsInfoOfLastPublicLinkShareUsingTheSharingApi(string $user): void {
		$response = $this->getLastShareInfo($user, "link");
		$this->setResponse($response);
		$this->pushToLastStatusCodesArrays();
	}

	/**
	 * @Then /^as "([^"]*)" the info about the last share by user "([^"]*)" with user "([^"]*)" should include$/
	 *
	 * @param string $requester
	 * @param string $sharer
	 * @param string $sharee
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	public function asLastShareInfoAboutUserSharingWithUserShouldInclude(
		string    $requester,
		string    $sharer,
		string    $sharee,
		TableNode $table
	) {
		$response = $this->getLastShareInfo($requester, "user");
		$this->theHTTPStatusCodeShouldBe(200, "", $response);
		$this->ocsContext->theOCSStatusCodeShouldBe("100,200", "", $response);
		$this->checkTheFieldsOfLastResponseToUser($sharer, $sharee, $table);
	}

	/**
	 * Get share data of specific share_id
	 *
	 * @param string $user
	 * @param string $share_id
	 * @param string|null $language
	 *
	 * @return ResponseInterface
	 */
	public function getShareData(string $user, string $share_id, ?string $language = null): ResponseInterface {
		$user = $this->getActualUsername($user);
		$url = $this->getSharesEndpointPath("/$share_id");
		$headers = [];
		if ($language !== null) {
			$headers['Accept-Language'] = $language;
		}
		return $this->ocsContext->sendRequestToOcsEndpoint(
			$user,
			"GET",
			$url,
			null,
			null,
			$headers
		);
	}

	/**
	 * @param string $user
	 *
	 * @return ResponseInterface
	 */
	public function getSharedWithMeShares(string $user): ResponseInterface {
		$user = $this->getActualUsername($user);
		$url = "/apps/files_sharing/api/v1/shares?shared_with_me=true";
		return $this->ocsContext->sendRequestToOcsEndpoint(
			$user,
			'GET',
			$url
		);
	}

	/**
	 * @When user :user gets all the shares shared with him/her using the sharing API
	 *
	 * @param string $user
	 *
	 * @return void
	 */
	public function userGetsAllTheSharesSharedWithHimUsingTheSharingApi(string $user): void {
		$this->setResponse($this->getSharedWithMeShares($user));
	}

	/**
	 * @Then as user :user the last share should include the following properties:
	 *
	 * @param string $user
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userGetsTheLastShareSharedWithHimUsingTheSharingApi(string $user, TableNode $table): void {
		$user = $this->getActualUsername($user);
		$this->verifyTableNodeRows($table, [], $this->shareResponseFields);
		$share_id = ($this->isUsingSharingNG())
		? $this->shareNgGetLastCreatedUserGroupShareID() : $this->getLastCreatedUserGroupShareId();

		$response = $this->getShareData($user, $share_id);
		$this->theHTTPStatusCodeShouldBe(
			200,
			"Error getting info of last share for user $user",
			$response
		);
		$this->ocsContext->assertOCSResponseIndicatesSuccess(
			__METHOD__ .
			' Error getting info of last share for user $user\n' .
			$this->ocsContext->getOCSResponseStatusMessage(
				$response
			) . '"',
			$response
		);

		$this->checkTheFields($user, $table, $response);
	}

	/**
	 * @When /^user "([^"]*)" gets the (|pending)\s?(user|group|user and group|public link) shares shared with (?:him|her) using the sharing API$/
	 *
	 * @param string $user
	 * @param string $pending
	 * @param string $shareType
	 *
	 * @return void
	 */
	public function userGetsFilteredSharesSharedWithHimUsingTheSharingApi(
		string $user,
		string $pending,
		string $shareType
	): void {
		$user = $this->getActualUsername($user);
		if ($pending === "pending") {
			$pendingClause = "&state=" . SharingHelper::SHARE_STATES['pending'];
		} else {
			$pendingClause = "";
		}
		if ($shareType === 'public link') {
			$shareType = 'public_link';
		}
		if ($shareType === 'user and group') {
			$rawShareTypes = SharingHelper::SHARE_TYPES['user'] . "," . SharingHelper::SHARE_TYPES['group'];
		} else {
			$rawShareTypes = SharingHelper::SHARE_TYPES[$shareType];
		}
		$response = $this->ocsContext->sendRequestToOcsEndpoint(
			$user,
			'GET',
			$this->getSharesEndpointPath(
				"?shared_with_me=true" . $pendingClause . "&share_types=" . $rawShareTypes
			)
		);
		$this->setResponse($response);
	}

	/**
	 * @When /^user "([^"]*)" gets all the shares shared with (?:him|her|them) that are received as (?:file|folder|entry) "([^"]*)" using the provisioning API$/
	 *
	 * @param string $user
	 * @param string $path
	 *
	 * @return void
	 */
	public function userGetsAllSharesSharedWithHimFromFileOrFolderUsingTheProvisioningApi(
		string $user,
		string $path
	): void {
		$user = $this->getActualUsername($user);
		$url = "/apps/files_sharing/api/"
			. "v$this->sharingApiVersion/shares?shared_with_me=true&path=$path";
		$response = $this->ocsContext->sendRequestToOcsEndpoint(
			$user,
			'GET',
			$url
		);
		$this->setResponse($response);
	}

	/**
	 * @param string $user
	 * @param string|null $endpointPath
	 *
	 * @return ResponseInterface
	 * @throws JsonException
	 * @throws GuzzleException
	 */
	public function getAllShares(
		string $user,
		?string $endpointPath = null
	): ResponseInterface {
		$user = $this->getActualUsername($user);
		return OcsApiHelper::sendRequest(
			$this->getBaseUrl(),
			$user,
			$this->getPasswordForUser($user),
			"GET",
			$this->getSharesEndpointPath($endpointPath),
			$this->getStepLineRef(),
			[],
			$this->ocsApiVersion
		);
	}

	/**
	 * @When /^user "([^"]*)" gets all shares shared by (?:him|her) using the sharing API$/
	 *
	 * @param string $user
	 *
	 * @return void
	 */
	public function userGetsAllSharesSharedByHimUsingTheSharingApi(string $user): void {
		$this->setResponse($this->getAllshares($user));
	}

	/**
	 * @When /^the administrator gets all shares shared by (?:him|her) using the sharing API$/
	 *
	 * @return void
	 */
	public function theAdministratorGetsAllSharesSharedByHimUsingTheSharingApi(): void {
		$this->setResponse($this->getAllShares($this->getAdminUsername()));
	}

	/**
	 * @When /^user "([^"]*)" gets the (user|group|user and group|public link) shares shared by (?:him|her) using the sharing API$/
	 *
	 * @param string $user
	 * @param string $shareType
	 *
	 * @return void
	 */
	public function userGetsFilteredSharesSharedByHimUsingTheSharingApi(string $user, string $shareType): void {
		if ($shareType === 'public link') {
			$shareType = 'public_link';
		}
		if ($shareType === 'user and group') {
			$rawShareTypes = SharingHelper::SHARE_TYPES['user'] . "," . SharingHelper::SHARE_TYPES['group'];
		} else {
			$rawShareTypes = SharingHelper::SHARE_TYPES[$shareType];
		}
		$this->setResponse($this->getAllShares($user, "?share_types=" . $rawShareTypes));
	}

	/**
	 * @When user :user gets all the shares of the file :path using the sharing API
	 *
	 * @param string $user
	 * @param string $path
	 *
	 * @return void
	 */
	public function userGetsAllTheSharesFromTheFileUsingTheSharingApi(string $user, string $path): void {
		$this->setResponse($this->getAllShares($user, "?path=$path"));
	}

	/**
	 * @Then /^the last share_id should be included in the response/
	 *
	 * @return void
	 * @throws Exception
	 */
	public function checkingLastShareIDIsIncluded(): void {
		$shareId = ($this->isUsingSharingNG())
		? $this->shareNgGetLastCreatedUserGroupShareID() : $this->getLastCreatedUserGroupShareId();
		if (!$this->isFieldInResponse('id', $shareId)) {
			Assert::fail(
				"Share id $shareId not found in response"
			);
		}
	}

	/**
	 * @Then /^the last share id should not be included in the response/
	 *
	 * @return void
	 * @throws Exception
	 */
	public function checkLastShareIDIsNotIncluded(): void {
		$shareId = $this->isUsingSharingNG()
		? $this->shareNgGetLastCreatedUserGroupShareID() : $this->getLastCreatedUserGroupShareId();
		if ($this->isFieldInResponse('id', $shareId, false)) {
			Assert::fail(
				"Share id $shareId has been found in response"
			);
		}
	}

	/**
	 *
	 * @param ResponseInterface $response
	 *
	 * @return void
	 */
	public function responseShouldNotContainAnyShareIds(ResponseInterface $response): void {
		$data = HttpRequestHelper::getResponseXml($response, __METHOD__)->data[0];
		$fieldIsSet = false;
		$receivedShareCount = 0;

		if (\count($data->element) > 0) {
			foreach ($data as $element) {
				if (isset($element->id)) {
					$fieldIsSet = true;
					$receivedShareCount += 1;
				}
			}
		} else {
			if (isset($data->id)) {
				$fieldIsSet = true;
				$receivedShareCount += 1;
			}
		}
		Assert::assertFalse(
			$fieldIsSet,
			"response contains $receivedShareCount share ids but should not contain any share ids"
		);
	}

	/**
	 * @Then user :user should not see the share id of the last share
	 *
	 * @param string $user
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userShouldNotSeeShareIdOfLastShare(string $user): void {
		$response = $this->getSharedWithMeShares($user);
		$this->theHTTPStatusCodeShouldBe(200, "", $response);
		$shareId = $this->isUsingSharingNG()
		? $this->shareNgGetLastCreatedUserGroupShareID() : $this->getLastCreatedUserGroupShareId();
		if ($this->isFieldInResponse('id', $shareId, false)) {
			Assert::fail(
				"Share id $shareId has been found in response"
			);
		}
	}

	/**
	 * @Then user :user should not have any received shares
	 *
	 * @param string $user
	 *
	 * @return void
	 */
	public function userShouldNotHaveAnyReceivedShares(string $user): void {
		$response = $this->getSharedWithMeShares($user);
		$this->theHTTPStatusCodeShouldBe(200, "", $response);
		$this->responseShouldNotContainAnyShareIds($response);
	}

	/**
	 * @Then /^the response should contain ([0-9]+) entries$/
	 *
	 * @param int $count
	 *
	 * @return void
	 */
	public function checkingTheResponseEntriesCount(int $count): void {
		$actualCount = \count(HttpRequestHelper::getResponseXml($this->response, __METHOD__)->data[0]);
		Assert::assertEquals(
			$count,
			$actualCount,
			"Expected that the response should contain '$count' entries but got '$actualCount' entries"
		);
	}

	/**
	 * @Then the fields of the last response to user :user should include
	 *
	 * @param string $user
	 * @param TableNode|null $body
	 *
	 * @return void
	 * @throws Exception
	 */
	public function checkFields(string $user, ?TableNode $body): void {
		$this->checkTheFields($user, $body);
	}

	/**
	 * @param string $user
	 * @param TableNode|null $body
	 * @param ResponseInterface|null $response
	 *
	 * @return void
	 */
	public function checkTheFields(string $user, ?TableNode $body, ?ResponseInterface $response = null): void {
		$response = $response ?? $this->getResponse();
		$data = HttpRequestHelper::getResponseXml($response, __METHOD__)->data[0];
		$this->verifyTableNodeColumnsCount($body, 2);
		$bodyRows = $body->getRowsHash();
		$userRelatedFieldNames = [
			"owner",
			"user",
			"uid_owner",
			"uid_file_owner",
			"share_with",
			"displayname_file_owner",
			"displayname_owner",
			"additional_info_owner",
			"additional_info_file_owner"
		];
		foreach ($bodyRows as $field => $value) {
			if (\in_array($field, $userRelatedFieldNames)) {
				$value = $this->substituteInLineCodes($value, $user);
			}
			$value = $this->getActualUsername($value);
			$value = $this->replaceValuesFromTable($field, $value);
			Assert::assertTrue(
				$this->isFieldInResponse($field, $value, true, $data),
				"$field doesn't have value '$value'",
			);
		}
	}

	/**
	 * @Then the fields of the last response to user :user and space :space should include
	 *
	 * @param string $user
	 * @param string $space
	 * @param TableNode|null $body
	 *
	 * @return void
	 * @throws Exception
	 */
	public function checkFieldsOfSpaceSharingResponse(string $user, string $space, ?TableNode $body): void {
		$this->verifyTableNodeColumnsCount($body, 2);

		$bodyRows = $body->getRowsHash();
		$userRelatedFieldNames = [
			"owner",
			"user",
			"uid_owner",
			"uid_file_owner",
			"share_with",
			"displayname_file_owner",
			"displayname_owner",
			"additional_info_owner",
			"additional_info_file_owner",
			"space_id"
		];

		$response = HttpRequestHelper::getResponseXml($this->response, __METHOD__);
		$this->addToCreatedPublicShares($response->data);
		foreach ($bodyRows as $field => $value) {
			if (\in_array($field, $userRelatedFieldNames)) {
				$value = $this->substituteInLineCodes(
					$value,
					$user,
					[],
					[
						[
							"code" => "%space_id%",
							"function" =>
								[$this->spacesContext, "getSpaceIdByName"],
							"parameter" => [$user, $space]
						],
					],
					null,
					null
				);
				if ($field === "uid_file_owner") {
					$value = (explode("$", $value))[1];
				}
				if ($field === "space_id") {
					$explodedSpaceId = explode("$", $value);
					$value = $explodedSpaceId[0] . "$" . $explodedSpaceId[1] . "!" . $explodedSpaceId[1];
				}
			}
			$value = $this->getActualUsername($value);
			$value = $this->replaceValuesFromTable($field, $value);

			Assert::assertTrue(
				$this->isFieldInResponse($field, $value, true, $this->getLastCreatedPublicShare()),
				"$field doesn't have value '$value'"
			);
		}
	}

	/**
	 * @Then /^the fields of the last response (?:to|about) user "([^"]*)" sharing with (?:user|group) "([^"]*)" should include$/
	 *
	 * @param string $sharer
	 * @param string $sharee
	 * @param TableNode|null $body
	 *
	 * @return void
	 * @throws Exception
	 */
	public function checkFieldsOfLastResponseToUser(string $sharer, string $sharee, ?TableNode $body): void {
		$this->checkTheFieldsOfLastResponseToUser($sharer, $sharee, $body);
	}

	/**
	 * @param string $sharer
	 * @param string $sharee
	 * @param TableNode|null $body
	 *
	 * @return void
	 */
	public function checkTheFieldsOfLastResponseToUser(string $sharer, string $sharee, ?TableNode $body): void {
		$this->verifyTableNodeColumnsCount($body, 2);
		$bodyRows = $body->getRowsHash();
		foreach ($bodyRows as $field => $value) {
			if (\in_array(
				$field,
				[
					"displayname_owner",
					 "displayname_file_owner",
					 "owner",
					 "uid_owner",
					 "uid_file_owner",
					  "additional_info_owner",
					  "additional_info_file_owner"
					  ]
			)
			) {
				$value = $this->substituteInLineCodes($value, $sharer);
			} elseif (\in_array(
				$field,
				["share_with", "share_with_displayname", "user", "share_with_additional_info"]
			)
			) {
				$value = $this->substituteInLineCodes($value, $sharee);
			}
			$value = $this->replaceValuesFromTable($field, $value);
			Assert::assertTrue(
				$this->isFieldInResponse($field, $value),
				"$field doesn't have value '$value'"
			);
		}
	}

	/**
	 *
	 * @return string
	 *
	 * @throws Exception
	 */
	public function getSharingAttributesFromLastResponse(): string {
		$responseXmlObject = HttpRequestHelper::getResponseXml($this->response, __METHOD__)->data[0];
		$actualAttributesElement = $responseXmlObject->xpath('//attributes');

		if ($actualAttributesElement) {
			$actualAttributes = (array) $actualAttributesElement[0];
			if (empty($actualAttributes)) {
				throw new Exception(
					"No data inside 'attributes' element in the last response."
				);
			}
			return $actualAttributes[0];
		}

		throw new Exception("No 'attributes' found inside the response of the last share.");
	}

	/**
	 * @Then the fields of the last response should not include
	 *
	 * @param TableNode|null $body
	 *
	 * @return void
	 * @throws Exception
	 */
	public function checkFieldsNotInResponse(?TableNode $body): void {
		$this->verifyTableNodeColumnsCount($body, 2);
		$bodyRows = $body->getRowsHash();

		foreach ($bodyRows as $field => $value) {
			$value = $this->replaceValuesFromTable($field, $value);
			Assert::assertFalse(
				$this->isFieldInResponse($field, $value, false),
				"$field has value $value but should not"
			);
		}
	}

	/**
	 * Returns shares of a file or folder as a SimpleXMLElement
	 *
	 * Note: the "single" SimpleXMLElement may contain one or more actual
	 * shares (to users, groups or public links etc.). If you access an item directly,
	 * for example, getShares()->id, then the value of "id" for the first element
	 * will be returned. To access all the elements, you can loop through the
	 * returned SimpleXMLElement with "foreach" - it will act like a PHP array
	 * of elements.
	 *
	 * @param string $user
	 * @param string $path
	 *
	 * @return SimpleXMLElement
	 */
	public function getShares(string $user, string $path): SimpleXMLElement {
		$response = OcsApiHelper::sendRequest(
			$this->getBaseUrl(),
			$user,
			$this->getPasswordForUser($user),
			"GET",
			$this->getSharesEndpointPath("?path=$path"),
			$this->getStepLineRef(),
			[],
			$this->ocsApiVersion
		);
		return HttpRequestHelper::getResponseXml($response, __METHOD__)->data->element;
	}

	/**
	 * @Then /^as user "([^"]*)" the public shares of (?:file|folder) "([^"]*)" should be$/
	 *
	 * @param string $user
	 * @param string $path
	 * @param TableNode|null $TableNode
	 *
	 * @return void
	 * @throws Exception
	 */
	public function checkPublicShares(string $user, string $path, ?TableNode $TableNode): void {
		$user = $this->getActualUsername($user);
		$response = $this->getShares($user, $path);

		$this->verifyTableNodeColumns($TableNode, ['path', 'permissions', 'name']);
		if ($TableNode instanceof TableNode) {
			$elementRows = $TableNode->getHash();

			foreach ($elementRows as $expectedElementsArray) {
				$nameFound = false;
				foreach ($response as $elementResponded) {
					if ((string) $elementResponded->name[0] === $expectedElementsArray['name']) {
						Assert::assertEquals(
							$expectedElementsArray['path'],
							(string) $elementResponded->path[0],
							__METHOD__
							. " Expected '{$expectedElementsArray['path']}' but got '"
							. $elementResponded->path[0]
							. "'"
						);
						Assert::assertEquals(
							$expectedElementsArray['permissions'],
							(string) $elementResponded->permissions[0],
							__METHOD__
							. " Expected '{$expectedElementsArray['permissions']}' but got '"
							. $elementResponded->permissions[0]
							. "'"
						);
						$nameFound = true;
						break;
					}
				}
				Assert::assertTrue(
					$nameFound,
					"Shared link name {$expectedElementsArray['name']} not found"
				);
			}
		}
	}

	/**
	 * @Then /^as user "([^"]*)" the (?:file|folder) "([^"]*)" should not have any shares$/
	 *
	 * @param string $user
	 * @param string $path
	 *
	 * @return void
	 * @throws Exception
	 */
	public function checkPublicSharesAreEmpty(string $user, string $path): void {
		$user = $this->getActualUsername($user);
		$response = $this->getShares($user, $path);
		//It shouldn't have public shares
		Assert::assertEquals(
			0,
			\count($response),
			__METHOD__
			. " As '$user', '$path' was expected to have no shares, but got '"
			. \count($response)
			. "' shares present"
		);
	}

	/**
	 * @param string $user
	 * @param string $path to share
	 * @param string $name of share
	 *
	 * @return string|null
	 */
	public function getPublicShareIDByName(string $user, string $path, string $name): ?string {
		$response = $this->getShares($user, $path);
		foreach ($response as $elementResponded) {
			if ((string) $elementResponded->name[0] === $name) {
				return (string) $elementResponded->id[0];
			}
		}
		return null;
	}

	/**
	 * @param string $user
	 * @param string $name
	 * @param string $path
	 *
	 * @return ResponseInterface
	 */
	public function deletePublicLinkShareUsingTheSharingApi(
		string $user,
		string $name,
		string $path
	): ResponseInterface {
		$user = $this->getActualUsername($user);
		$share_id = $this->getPublicShareIDByName($user, $path, $name);
		$url = $this->getSharesEndpointPath("/$share_id");
		return $this->ocsContext->sendRequestToOcsEndpoint(
			$user,
			"DELETE",
			$url
		);
	}

	/**
	 * @When /^user "([^"]*)" deletes public link share named "([^"]*)" in (?:file|folder) "([^"]*)" using the sharing API$/
	 *
	 * @param string $user
	 * @param string $name
	 * @param string $path
	 *
	 * @return void
	 */
	public function userDeletesPublicLinkShareNamedUsingTheSharingApi(
		string $user,
		string $name,
		string $path
	): void {
		$response = $this->deletePublicLinkShareUsingTheSharingApi(
			$user,
			$name,
			$path
		);
		$this->setResponse($response);
	}

	/**
	 * @param string $user
	 * @param string $action
	 * @param string $share
	 * @param string $offeredBy
	 * @param string|null $state
	 *
	 * @return ResponseInterface
	 */
	public function reactToShareOfferedBy(
		string $user,
		string $action,
		string $share,
		string $offeredBy,
		?string $state = ''
	): ResponseInterface {
		$user = $this->getActualUsername($user);
		$offeredBy = $this->getActualUsername($offeredBy);

		$response = $this->getAllSharesSharedWithUser($user);
		$shareId = null;
		foreach ($response as $shareElement) {
			// SharingHelper::SHARE_STATES has the mapping between the words for share states
			// like "accepted", "pending",... and the integer constants 0, 1,... that are in
			// the "state" field of the share data.
			if ($state === '') {
				// Any share state is OK
				$matchesShareState = true;
			} else {
				$requiredStateCode = SharingHelper::SHARE_STATES[$state];
				if ($shareElement['state'] === $requiredStateCode) {
					$matchesShareState = true;
				} else {
					$matchesShareState = false;
				}
			}

			if ($matchesShareState
				&& (string) $shareElement['uid_owner'] === $offeredBy
				&& (string) $shareElement['path'] === $share
			) {
				$shareId = (string) $shareElement['id'];
				break;
			}
		}
		Assert::assertNotNull(
			$shareId,
			__METHOD__ . " could not find share $share, offered by $offeredBy to $user"
		);
		$url = "/apps/files_sharing/api/v$this->sharingApiVersion" .
			"/shares/pending/$shareId";
		if (\substr($action, 0, 7) === "decline") {
			$httpRequestMethod = "DELETE";
		} else {
			// do a POST to accept the share
			$httpRequestMethod = "POST";
		}

		return $this->ocsContext->sendRequestToOcsEndpoint(
			$user,
			$httpRequestMethod,
			$url
		);
	}

	/**
	 * @When /^user "([^"]*)" (declines|accepts) share "([^"]*)" offered by user "([^"]*)" using the sharing API$/
	 *
	 * @param string $user
	 * @param string $action
	 * @param string $share
	 * @param string $offeredBy
	 * @param string|null $state specify 'accepted', 'pending', 'rejected' or 'declined' to only consider shares in that state
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userReactsToShareOfferedBy(
		string $user,
		string $action,
		string $share,
		string $offeredBy,
		?string $state = ''
	): void {
		$response = $this->reactToShareOfferedBy(
			$user,
			$action,
			$share,
			$offeredBy
		);
		$this->setResponse($response);
		$this->pushToLastStatusCodesArrays();
	}

	/**
	 * @When /^user "([^"]*)" (declines|accepts) the already (?:accepted|declined) share "([^"]*)" offered by user "([^"]*)" using the sharing API$/
	 *
	 * @param string $user
	 * @param string $action
	 * @param string $share
	 * @param string $offeredBy
	 * @param string|null $state specify 'accepted', 'pending', 'rejected' or 'declined' to only consider shares in that state
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userAcceptsTheAlreadyAcceptedShareOfferedByUsingTheSharingApi(
		string $user,
		string $action,
		string $share,
		string $offeredBy,
		?string $state = ''
	): void {
		$response = $this->reactToShareOfferedBy(
			$user,
			$action,
			"/Shares/" . \trim($share, "/"),
			$offeredBy
		);
		$this->setResponse($response);
		$this->pushToLastStatusCodesArrays();
	}

	/**
	 * @When /^user "([^"]*)" (declines|accepts) the following shares offered by user "([^"]*)" using the sharing API$/
	 *
	 * @param string $user
	 * @param string $action
	 * @param string $offeredBy
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userReactsToTheFollowingSharesOfferedBy(
		string $user,
		string $action,
		string $offeredBy,
		TableNode $table
	): void {
		$this->verifyTableNodeColumns($table, ["path"]);
		$paths = $table->getHash();

		foreach ($paths as $share) {
			$response = $this->reactToShareOfferedBy(
				$user,
				$action,
				$share["path"],
				$offeredBy
			);
			$this->setResponse($response);
			$this->pushToLastStatusCodesArrays();
		}
		$this->pushToLastStatusCodesArrays();
	}

	/**
	 * @When /^user "([^"]*)" (declines|accepts) share with ID "([^"]*)" using the sharing API$/
	 *
	 * @param string $user
	 * @param string $action
	 * @param string $share_id
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userReactsToShareWithShareIDOfferedBy(string $user, string $action, string $share_id): void {
		$user = $this->getActualUsername($user);

		$shareId = $this->substituteInLineCodes($share_id, $user);

		$url = "/apps/files_sharing/api/v$this->sharingApiVersion" .
			"/shares/pending/$shareId";
		if (\substr($action, 0, 7) === "decline") {
			$httpRequestMethod = "DELETE";
		} else {
			// do a POST to accept the share
			$httpRequestMethod = "POST";
		}

		$response = $this->ocsContext->sendRequestToOcsEndpoint(
			$user,
			$httpRequestMethod,
			$url
		);
		$this->setResponse($response);
	}

	/**
	 * @Given /^user "([^"]*)" has (declined|accepted) share "([^"]*)" offered by user "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $action
	 * @param string $share
	 * @param string $offeredBy
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userHasReactedToShareOfferedBy(
		string $user,
		string $action,
		string $share,
		string $offeredBy
	): void {
		$response = $this->reactToShareOfferedBy(
			$user,
			$action,
			$share,
			$offeredBy
		);
		if ($action === 'declined') {
			$actionText = 'decline';
		} else {
			$actionText = 'accept';
		}
		$this->theHTTPStatusCodeShouldBe(
			200,
			__METHOD__ . " could not $actionText share $share to $user by $offeredBy",
			$response
		);
		$this->emptyLastHTTPStatusCodesArray();
		$this->emptyLastOCSStatusCodesArray();
	}

	/**
	 * @Then /^user "([^"]*)" should be able to (decline|accept) pending share "([^"]*)" offered by user "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $action
	 * @param string $share
	 * @param string $offeredBy
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userShouldBeAbleToAcceptShareOfferedBy(
		string $user,
		string $action,
		string $share,
		string $offeredBy
	) {
		if ($action === 'accept') {
			$response = $this->reactToShareOfferedBy($user, 'accepts', $share, $offeredBy, 'pending');
			$this->theHTTPStatusCodeShouldBe(
				200,
				__METHOD__ . " could not accept the pending share $share to $user by $offeredBy",
				$response
			);
			$this->ocsContext->theOCSStatusCodeShouldBe("100,200", "", $response);
		} elseif ($action === 'decline') {
			$response = $this->reactToShareOfferedBy($user, 'declined', $share, $offeredBy);
			$this->theHTTPStatusCodeShouldBe(
				200,
				__METHOD__ . " could not decline share $share to $user by $offeredBy",
				$response
			);
			$this->ocsContext->theOCSStatusCodeShouldBe("100,200", "", $response);
		}
	}

	/**
	 *
	 * @Then /^the sharing API should report to user "([^"]*)" that these shares are in the (pending|accepted|declined) state$/
	 *
	 * @param string $user
	 * @param string $state
	 * @param TableNode $table table with headings that correspond to the attributes
	 *                         of the share e.g. "|path|uid_owner|"
	 *
	 * @return void
	 * @throws Exception
	 */
	public function assertSharesOfUserAreInState(string $user, string $state, TableNode $table): void {
		$this->verifyTableNodeColumns($table, ["path"], $this->shareResponseFields);
		$usersShares = $this->getAllSharesSharedWithUser($user, $state);
		foreach ($table as $row) {
			$found = false;
			//the API returns the path without trailing slash, but we want to
			//be able to accept leading and/or trailing slashes in the step definition
			$row['path'] = "/" . \trim($row['path'], "/");
			foreach ($usersShares as $share) {
				try {
					Assert::assertArrayHasKey('path', $share);
					Assert::assertEquals($row['path'], $share['path']);
					$found = true;
					break;
				} catch (ExpectationFailedException $e) {
				}
			}
			if (!$found) {
				Assert::fail(
					"could not find the share with this attributes " .
					\print_r($row, true)
				);
			}
		}
	}

	/**
	 *
	 * @Then /^the sharing API should report to user "([^"]*)" that no shares are in the (pending|accepted|declined) state$/
	 *
	 * @param string $user
	 * @param string $state
	 *
	 * @return void
	 * @throws Exception
	 */
	public function assertNoSharesOfUserAreInState(string $user, string $state): void {
		$usersShares = $this->getAllSharesSharedWithUser($user, $state);
		Assert::assertEmpty(
			$usersShares,
			"user has " . \count($usersShares) . " share(s) in the $state state"
		);
	}

	/**
	 * @param string $sharer
	 * @param string $path
	 * @param string $sharee
	 *
	 * @return ResponseInterface
	 */
	public function unshareResourceSharedTo(string $sharer, string $path, string $sharee): ResponseInterface {
		$sharer = $this->getActualUsername($sharer);
		$sharee = $this->getActualUsername($sharee);

		$response = $this->getShares($sharer, "$path&share_types=0");
		$shareId = null;
		foreach ($response as $shareElement) {
			if ((string)$shareElement->share_with[0] === $sharee) {
				$shareId = (string) $shareElement->id;
				break;
			}
		}
		Assert::assertNotNull(
			$shareId,
			__METHOD__ . " could not find share, offered by $sharer to $sharee"
		);

		return $this->ocsContext->sendRequestToOcsEndpoint(
			$sharer,
			'DELETE',
			'/apps/files_sharing/api/v' . $this->sharingApiVersion . '/shares/' . $shareId
		);
	}

	/**
	 * @When /^user "([^"]*)" unshares (?:folder|file|entity) "([^"]*)" shared to "([^"]*)"$/
	 *
	 * @param string $sharer
	 * @param string $path
	 * @param string $sharee
	 *
	 * @return void
	 * @throws JsonException
	 */
	public function userUnsharesResourceSharedTo(string $sharer, string $path, string $sharee): void {
		$response = $this->unshareResourceSharedTo($sharer, $path, $sharee);
		$this->setResponse($response);
	}

	/**
	 * @Then the sharing API should report that no shares are shared with user :user
	 *
	 * @param string $user
	 *
	 * @return void
	 * @throws Exception
	 */
	public function assertThatNoSharesAreSharedWithUser(string $user): void {
		$usersShares = $this->getAllSharesSharedWithUser($user);
		Assert::assertEmpty(
			$usersShares,
			"user has " . \count($usersShares) . " share(s)"
		);
	}

	/**
	 * @When user :user gets share with id :share using the sharing API
	 *
	 * @param string $user
	 * @param string $share_id
	 *
	 * @return ResponseInterface|null
	 */
	public function userGetsTheLastShareWithTheShareIdUsingTheSharingApi(
		string $user,
		string $share_id
	): ?ResponseInterface {
		$user = $this->getActualUsername($user);
		$share_id = $this->substituteInLineCodes($share_id, $user);
		$url = $this->getSharesEndpointPath("/$share_id");

		$this->response = OcsApiHelper::sendRequest(
			$this->getBaseUrl(),
			$user,
			$this->getPasswordForUser($user),
			"GET",
			$url,
			$this->getStepLineRef(),
			[],
			$this->ocsApiVersion
		);
		return $this->response;
	}

	/**
	 *
	 * @param string $user
	 * @param string|null $state pending|accepted|declined|rejected|all
	 *
	 * @return array of shares that are shared with this user
	 * @throws Exception
	 */
	private function getAllSharesSharedWithUser(string $user, ?string $state = "all"): array {
		switch ($state) {
			case 'pending':
			case 'accepted':
			case 'declined':
			case 'rejected':
				$stateCode = SharingHelper::SHARE_STATES[$state];
				break;
			case 'all':
				$stateCode = "all";
				break;
			default:
				throw new InvalidArgumentException(
					__METHOD__ . ' invalid "state" given'
				);
		}
		$url = $this->getSharesEndpointPath("?format=json&shared_with_me=true&state=$stateCode");
		$response = $this->ocsContext->sendRequestToOcsEndpoint(
			$user,
			"GET",
			$url
		);
		if ($response->getStatusCode() !== 200) {
			throw new Exception(
				__METHOD__ . " could not retrieve information about shares"
			);
		}
		$result = $response->getBody()->getContents();
		$usersShares = \json_decode($result, true);
		if (!\is_array($usersShares)) {
			throw new Exception(
				__METHOD__ . " API result about shares is not valid JSON"
			);
		}
		return $usersShares['ocs']['data'];
	}

	/**
	 * Send request for preview of a file in a public link
	 *
	 * @param string $fileName
	 * @param string $token
	 *
	 * @return ResponseInterface
	 */
	public function getPublicPreviewOfFile(string $fileName, string $token): ResponseInterface {
		$baseUrl = $this->getBaseUrl();
		$davPath = WebdavHelper::getDavPath($this->getDavPathVersion(), $token, "public-files");
		$url = "$baseUrl/$davPath/$fileName?preview=1";
		return HttpRequestHelper::get(
			$url,
			$this->getStepLineRef()
		);
	}

	/**
	 * @When the public accesses the preview of file :path from the last shared public link using the sharing API
	 *
	 * @param string $path
	 *
	 * @return void
	 */
	public function thePublicAccessesThePreviewOfTheSharedFileUsingTheSharingApi(string $path): void {
		$token = ($this->isUsingSharingNG())
		? $this->shareNgGetLastCreatedLinkShareToken() : $this->getLastCreatedPublicShareToken();
		$response = $this->getPublicPreviewOfFile($path, $token);
		$this->setResponse($response);
		$this->pushToLastStatusCodesArrays();
	}

	/**
	 * @When the public accesses the preview of the following files from the last shared public link using the sharing API
	 *
	 * @param TableNode $table
	 *
	 * @throws Exception
	 * @return void
	 */
	public function thePublicAccessesThePreviewOfTheFollowingSharedFileUsingTheSharingApi(
		TableNode $table
	): void {
		$this->verifyTableNodeColumns($table, ["path"]);
		$paths = $table->getHash();
		$this->emptyLastHTTPStatusCodesArray();
		$this->emptyLastOCSStatusCodesArray();
		foreach ($paths as $path) {
			$token = ($this->isUsingSharingNG())
			? $this->shareNgGetLastCreatedLinkShareToken() : $this->getLastCreatedPublicShareToken();
			$response = $this->getPublicPreviewOfFile($path["path"], $token);
			$this->setResponse($response);
			$this->pushToLastStatusCodesArrays();
		}
	}

	/**
	 * @param string $user
	 * @param string $shareServer
	 * @param string|null $password
	 *
	 * @return ResponseInterface
	 */
	public function saveLastSharedPublicLinkShare(
		string $user,
		string $shareServer,
		?string $password = ""
	): ResponseInterface {
		$user = $this->getActualUsername($user);
		$userPassword = $this->getPasswordForUser($user);

		$shareData = $this->getLastCreatedPublicShare();
		$owner = (string) $shareData->uid_owner;
		$name = $this->encodePath((string) $shareData->file_target);
		$name = \trim($name, "/");
		$ownerDisplayName = (string) $shareData->displayname_owner;
		$token = (string) $shareData->token;

		if (\strtolower($shareServer) == "remote") {
			$remote = $this->getRemoteBaseUrl();
		} else {
			$remote = $this->getLocalBaseUrl();
		}

		$body['remote'] = $remote;
		$body['token'] = $token;
		$body['owner'] = $owner;
		$body['ownerDisplayName'] = $ownerDisplayName;
		$body['name'] = $name;
		$body['password'] = $password;

		Assert::assertNotNull(
			$token,
			__METHOD__ . " could not find any public share"
		);

		$url = $this->getBaseUrl() . "/index.php/apps/files_sharing/external";

		$response = HttpRequestHelper::post(
			$url,
			$this->getStepLineRef(),
			$user,
			$userPassword,
			null,
			$body
		);
		return $response;
	}

	/**
	 * @param string $user
	 * @param string $path
	 * @param string $type
	 * @param int $permissions
	 *
	 * @return array
	 */
	public function preparePublicQuickLinkPayload(
		string $user,
		string $path,
		string $type,
		int $permissions = 1
	): array {
		return [
			"permissions" => $permissions,
			"expireDate" => "",
			"shareType" => 3,
			"itemType" => $type,
			"itemSource" => $this->getFileIdForPath($user, $path),
			"name" => "Public quick link",
			"attributes" =>  [
				[
					"scope" => "files_sharing",
					"key" => "isQuickLink",
					"value" => true
				]
			],
			"path" => $path
		];
	}

	/**
	 * Expires last created public link share using the testing API
	 *
	 * @return void
	 * @throws ResponseInterface
	 */
	public function expireLastCreatedPublicLinkShare(): ResponseInterface {
		$shareId = (string) $this->getLastCreatedPublicShare()->id;
		return $this->expireShare($shareId);
	}

	/**
	 * Expires a share using the testing API
	 *
	 * @param string|null $shareId optional share id, if null then expire the last share that was created.
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function expireShare(string $shareId = null): ResponseInterface {
		$adminUser = $this->getAdminUsername();
		if ($shareId === null) {
			$shareId = $this->getLastCreatedUserGroupShareId();
		}
		$response = OcsApiHelper::sendRequest(
			$this->getBaseUrl(),
			$adminUser,
			$this->getAdminPassword(),
			'POST',
			"/apps/testing/api/v1/expire-share/$shareId",
			$this->getStepLineRef(),
			[],
			$this->getOcsApiVersion()
		);
		return $response;
	}

	/**
	 * replace values from table
	 *
	 * @param string $field
	 * @param string $value
	 *
	 * @return string
	 */
	public function replaceValuesFromTable(string $field, string $value): string {
		if (\substr($field, 0, 10) === "share_with") {
			$value = \str_replace(
				"REMOTE",
				$this->getRemoteBaseUrl(),
				$value
			);
			$value = \str_replace(
				"LOCAL",
				$this->getLocalBaseUrl(),
				$value
			);
		}
		if (\substr($field, 0, 6) === "remote") {
			$value = \str_replace(
				"REMOTE",
				$this->getRemoteBaseUrl(),
				$value
			);
			$value = \str_replace(
				"LOCAL",
				$this->getLocalBaseUrl(),
				$value
			);
		}
		if ($field === "permissions") {
			if (\is_string($value) && !\is_numeric($value)) {
				$value = $this->splitPermissionsString($value);
			}
			$value = (string)SharingHelper::getPermissionSum($value);
		}
		if ($field === "share_type") {
			$value = (string)SharingHelper::getShareType($value);
		}
		return $value;
	}
}
