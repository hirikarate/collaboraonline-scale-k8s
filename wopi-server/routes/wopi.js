var express = require('express');
const fs = require('fs');
const path = require('path');
const { globSync } = require('glob');

var router = express.Router();
const dataDir = path.join(__dirname, '..', 'data');

/* *
 *  wopi CheckFileInfo endpoint
 *
 *  Returns info about the file with the given document id.
 *  The response has to be in JSON format and at a minimum it needs to include
 *  the file name and the file size.
 *  The CheckFileInfo wopi endpoint is triggered by a GET request at
 *  https://HOSTNAME/wopi/files/<document_id>
 */
router.get('/files/:fileId', function(req, res) {
	const fileId = req.params.fileId;
	const pattern = path.join(dataDir, fileId + '.*').replaceAll('\\', '/');
	console.log('glob pattern: ' + pattern);

	try {
		const files = globSync(pattern);
		console.log('glob files: ' + files);
		if (!files || files.length === 0) {
			res.status(404).send('File not found');
			return;
		}
		const filePath = files[0];
		try {
			const data = fs.statSync(filePath);
			res.json({
				BaseFileName: path.basename(filePath),
				Size: data.size,
				UserId: 1,
				UserCanWrite: true
			});
		} catch (err) {
			console.error('Error reading file:', err);
			res.status(500).send('Internal server error');
		}
	} catch (err) {
		console.error('Error searching for file:', err);
		res.status(500).send('Internal server error');
	}
});

/* *
 *  wopi GetFile endpoint
 *
 *  Given a request access token and a document id, sends back the contents of the file.
 *  The GetFile wopi endpoint is triggered by a request with a GET verb at
 *  https://HOSTNAME/wopi/files/<document_id>/contents
 */
router.get('/files/:fileId/contents', function(req, res) {
	// we just return the content of a fake text file
	// in a real case you should use the file id
	// for retrieving the file from the storage and
	// send back the file content as response
	// var fileContent = 'Hello world!';
	// res.send(fileContent);

	const fileId = req.params.fileId;
	const pattern = path.join(dataDir, fileId + '.*').replaceAll('\\', '/');
	console.log('glob pattern: ' + pattern);

	try {
		const files = globSync(pattern);
		console.log('glob files: ' + files);
		if (!files || files.length === 0) {
			res.status(404).send('File not found');
			return;
		}
		const filePath = files[0];
		try {
			const data = fs.readFileSync(filePath);
			res.send(data);
		} catch (err) {
			console.error('Error reading file:', err);
			res.status(500).send('Internal server error');
		}
	} catch (err) {
		console.error('Error searching for file:', err);
		res.status(500).send('Internal server error');
	}
});

/* *
 *  wopi PutFile endpoint
 *
 *  Given a request access token and a document id, replaces the files with the POST request body.
 *  The PutFile wopi endpoint is triggered by a request with a POST verb at
 *  https://HOSTNAME/wopi/files/<document_id>/contents
 */
router.post('/files/:fileId/contents', function(req, res) {
	const fileId = req.params.fileId;
	const pattern = path.join(dataDir, fileId + '.*').replaceAll('\\', '/');
	console.log('PutFile - glob pattern: ' + pattern);

	try {
		// Find the existing file to get its extension
		const files = globSync(pattern);
		console.log('PutFile - glob files: ' + files);
		
		if (!files || files.length === 0) {
			res.status(404).send('File not found');
			return;
		}

		const filePath = files[0];

		// Check if we have file content in the request body
		if (!req.body || req.body.length === 0) {
			console.log('PutFile - No file content received');
			res.status(400).send('No file content provided');
			return;
		}
		
		try {
			// Write the new content to the file
			fs.writeFileSync(filePath, req.body);
			console.log('PutFile - Successfully saved file:', filePath);
			console.log('PutFile - File size:', req.body.length, 'bytes');
			
			// Return success status
			res.sendStatus(200);
		} catch (writeErr) {
			console.error('PutFile - Error writing file:', writeErr);
			res.status(500).send('Error saving file');
		}
	} catch (err) {
		console.error('PutFile - Error searching for file:', err);
		res.status(500).send('Internal server error');
	}
});

module.exports = router;
