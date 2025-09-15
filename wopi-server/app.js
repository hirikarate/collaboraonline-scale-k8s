var express = require('express');
var path = require('path');
var logger = require('morgan');
var bodyParser = require('body-parser');
var cors = require('cors');

var indexRouter = require('./routes/index');
var wopiRouter = require('./routes/wopi');

// maximum request body size handled by the bodyParser package
// increase it if you need to handle larger files
var maxDocumentSize = '50mb';

var app = express();

// Enable CORS for all origins
app.use(cors({
  origin: '*',
  methods: ['GET', 'POST', 'PUT', 'DELETE', 'OPTIONS'],
  allowedHeaders: ['Content-Type', 'Authorization', 'X-Requested-With'],
  credentials: false
}));

app.use(logger('dev'));
app.use(express.json());
app.use(express.urlencoded({ extended: false }));
app.use(bodyParser.raw({limit: maxDocumentSize}));
app.use(express.static(path.join(__dirname, 'public')));

app.use('/', indexRouter);
app.use('/wopi', wopiRouter);

module.exports = app;
