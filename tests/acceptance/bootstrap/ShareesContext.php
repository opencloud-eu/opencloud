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

use Behat\Behat\Context\Context;
use Behat\Behat\Hook\Scope\BeforeScenarioScope;
use Behat\Gherkin\Node\TableNode;
use Psr\Http\Message\ResponseInterface;
use PHPUnit\Framework\Assert;
use TestHelpers\BehatHelper;
use TestHelpers\HttpRequestHelper;
use Behat\Step\Then;
use Behat\Step\When;

require_once 'bootstrap.php';

/**
 * Sharees context.
 */
class ShareesContext implements Context {
	private FeatureContext $featureContext;
	private OCSContext $ocsContext;

	/**
	 *
	 * @param string $user
	 * @param TableNode $body
	 *
	 * @return void
	 */
	#[When('user :user gets the sharees using the sharing API with parameters')]
	public function userGetsTheShareesWithParameters(string $user, TableNode $body): void {
		$this->featureContext->setResponse(
			$this->getShareesWithParameters(
				$user,
				$body
			)
		);
	}

	/**
	 *
	 * @param string $shareeType
	 * @param TableNode $shareesList
	 *
	 * @return void
	 * @throws Exception
	 */
	#[Then('the :shareeType sharees returned should be')]
	public function theShareesReturnedShouldBe(string $shareeType, TableNode $shareesList): void {
		$this->featureContext->verifyTableNodeColumnsCount($shareesList, 4);
		$sharees = $shareesList->getRows();
		$respondedArray = $this->getArrayOfShareesResponded(
			$this->featureContext->getResponse(),
			$shareeType
		);
		Assert::assertEquals(
			$sharees,
			$respondedArray,
			"Returned sharees do not match the expected ones. See the differences below."
		);
	}

	/**
	 *
	 * @param string $shareeType
	 *
	 * @return void
	 */
	#[Then('the :shareeType sharees returned should be empty')]
	public function theShareesReturnedShouldBeEmpty(string $shareeType): void {
		$respondedArray = $this->getArrayOfShareesResponded(
			$this->featureContext->getResponse(),
			$shareeType
		);
		if (isset($respondedArray[0])) {
			// [0] is display name and [2] is user or group id
			$firstEntry = $respondedArray[0][0] . " (" . $respondedArray[0][2] . ")";
		} else {
			$firstEntry = "";
		}

		Assert::assertEmpty(
			$respondedArray,
			"'$shareeType' array should be empty, but it starts with $firstEntry"
		);
	}

	/**
	 * @param ResponseInterface $response
	 * @param string $shareeType
	 *
	 * @return array
	 * @throws Exception
	 */
	public function getArrayOfShareesResponded(
		ResponseInterface $response,
		string $shareeType
	): array {
		$elements = HttpRequestHelper::getResponseXml($response, __METHOD__)->data;
		$elements = \json_decode(\json_encode($elements), true);
		if (\strpos($shareeType, 'exact ') === 0) {
			$elements = $elements['exact'];
			$shareeType = \substr($shareeType, 6);
		}

		Assert::assertArrayHasKey(
			$shareeType,
			$elements,
			__METHOD__ . " The sharees response does not have key '$shareeType'"
		);

		$sharees = [];
		foreach ($elements[$shareeType] as $element) {
			if (\is_int(\key($element))) {
				// this is a list of elements instead of just one item,
				// so return the list
				foreach ($element as $innerItem) {
					$sharees[] = [
						$innerItem['label'],
						$innerItem['value']['shareType'],
						$innerItem['value']['shareWith'],
						$innerItem['value']['shareWithAdditionalInfo']
					];
				}
			} else {
				$sharees[] = [
					$element['label'],
					$element['value']['shareType'],
					$element['value']['shareWith'],
					$element['value']['shareWithAdditionalInfo']
				];
			}
		}
		return $sharees;
	}

	/**
	 * @param string $user
	 * @param TableNode $body
	 *
	 * @return ResponseInterface
	 */
	public function getShareesWithParameters(string $user, TableNode $body): ResponseInterface {
		$user = $this->featureContext->getActualUsername($user);
		$url = '/apps/files_sharing/api/v1/sharees';
		$this->featureContext->verifyTableNodeColumnsCount($body, 2);
		$parameters = [];
		foreach ($body->getRowsHash() as $key => $value) {
			$parameters[] = "$key=$value";
		}
		if (!empty($parameters)) {
			$url .= '?' . \implode('&', $parameters);
		}

		return $this->ocsContext->sendRequestToOcsEndpoint(
			$user,
			'GET',
			$url
		);
	}

	/**
	 * This will run before EVERY scenario.
	 * It will set the properties for this object.
	 *
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
		$this->ocsContext = BehatHelper::getContext($scope, $environment, 'OCSContext');
	}
}
