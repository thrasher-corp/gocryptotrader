'use strict';

// Declare app level module which depends on views, and components
angular.module('gocryptoweb', [
  'ngRoute',
  'gocryptoweb.controllers'
]).
config(['$locationProvider', '$routeProvider', function($locationProvider, $routeProvider) {
  $locationProvider.hashPrefix('!');

  $routeProvider.otherwise({redirectTo: '/view1'});
}]);
