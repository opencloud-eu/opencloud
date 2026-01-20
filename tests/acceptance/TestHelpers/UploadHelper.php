<?php declare(strict_types=1);
/**
 * @author Artur Neumann <artur@jankaritech.com>
 * @copyright Copyright (c) 2017 Artur Neumann artur@jankaritech.com
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
namespace TestHelpers;

use GuzzleHttp\Exception\GuzzleException;
use PHPUnit\Framework\Assert;
use Psr\Http\Message\ResponseInterface;

/**
 * Helper for Uploads
 *
 * @author Artur Neumann <artur@jankaritech.com>
 *
 */
class UploadHelper extends Assert {
	/**
	 *
	 * @param string|null $baseUrl URL of OpenCloud
	 * @param string|null $user
	 * @param string|null $password
	 * @param string|null $source
	 * @param string|null $destination
	 * @param string|null $xRequestId
	 * @param array|null $headers
	 * @param int|null $davPathVersionToUse (1|2)
	 * @param bool $doChunkUpload
	 * @param int|null $noOfChunks how many chunks to upload
	 * @param bool|null $isGivenStep
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function upload(
		?string $baseUrl,
		?string $user,
		?string $password,
		?string $source,
		?string $destination,
		?string $xRequestId = '',
		?array $headers = [],
		?int $davPathVersionToUse = 1,
		bool $doChunkUpload = false,
		?int $noOfChunks = 1,
		?bool $isGivenStep = false,
	): ResponseInterface {
		if (!$doChunkUpload) {
			$data = \file_get_contents($source);
			return WebDavHelper::makeDavRequest(
				$baseUrl,
				$user,
				$password,
				"PUT",
				$destination,
				$headers,
				null,
				$xRequestId,
				$data,
				$davPathVersionToUse,
				"files",
				null,
				"basic",
				false,
				0,
				null,
				[],
				null,
				$isGivenStep
			);
		}

		//prepare chunking
		$chunks = self::chunkFile($source, $noOfChunks);
		$chunkingId = 'chunking-' . \rand(1000, 9999);
		$result = null;

		//upload chunks
		foreach ($chunks as $index => $chunk) {
			$filename = $destination . "-" . $chunkingId . "-" . \count($chunks) . '-' . $index;
			$result = WebDavHelper::makeDavRequest(
				$baseUrl,
				$user,
				$password,
				"PUT",
				$filename,
				$headers,
				null,
				$xRequestId,
				$chunk,
				$davPathVersionToUse,
				"files",
				null,
				"basic",
				false,
				0,
				null,
				[],
				null,
				$isGivenStep
			);
			if ($result->getStatusCode() >= 400) {
				return $result;
			}
		}

		self::assertNotNull($result, __METHOD__ . " chunking was requested but no upload was done.");
		return $result;
	}

	/**
	 * cut the file in multiple chunks
	 * returns an array of chunks with the content of the file
	 *
	 * @param string|null $file
	 * @param int|null $noOfChunks
	 *
	 * @return array $string
	 */
	public static function chunkFile(?string $file, ?int $noOfChunks = 1): array {
		$size = \filesize($file);
		$chunkSize = \ceil($size / $noOfChunks);
		$chunks = [];
		$fp = \fopen($file, 'r');
		while (!\feof($fp) && \ftell($fp) < $size) {
			$chunks[] = \fread($fp, (int)$chunkSize);
		}
		\fclose($fp);
		if (\count($chunks) === 0) {
			// chunk an empty file
			$chunks[] = '';
		}
		return $chunks;
	}

	/**
	 * creates a File with a specific size
	 *
	 * @param string|null $name full path of the file to create
	 * @param int|null $size
	 *
	 * @return void
	 */
	public static function createFileSpecificSize(?string $name, ?int $size): void {
		if (\file_exists($name)) {
			\unlink($name);
		}
		$file = \fopen($name, 'w');
		\fseek($file, \max($size - 1, 0), SEEK_CUR);
		if ($size) {
			\fwrite($file, 'a'); // write a dummy char at SIZE position
		}
		\fclose($file);
		self::assertEquals(
			1,
			\file_exists($name)
		);
		self::assertEquals(
			$size,
			\filesize($name)
		);
	}

	/**
	 * creates a File with a specific text content
	 *
	 * @param string|null $name full path of the file to create
	 * @param string|null $text
	 *
	 * @return void
	 */
	public static function createFileWithText(?string $name, ?string $text): void {
		$file = \fopen($name, 'w');
		\fwrite($file, $text);
		\fclose($file);
		self::assertEquals(
			1,
			\file_exists($name)
		);
	}

	/**
	 * get the path of the acceptance tests directory
	 *
	 * @return string
	 */
	public static function getAcceptanceTestsDir(): string {
		return \dirname(__FILE__) . "/../";
	}

	/**
	 * get the path of the filesForUpload directory
	 *
	 * @return string
	 */
	public static function getFilesForUploadDir(): string {
		return \dirname(__FILE__) . "/../filesForUpload/";
	}
}
