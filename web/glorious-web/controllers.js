'use strict';

/* Controllers */

angular.module('gocryptoweb.controllers', []).
  controller('HomeController', function ($scope, $http) {
    $scope.working = true;
    $http({
      method: 'GET',
      url: '/data/all-enabled-currencies'
    }).
    success(function (data, status, headers, config) {
      $scope.exchanges = data.data.exchanges;
    }).
    error(function (data, status, headers, config) {
      console.log('error');
    });

  }).
  controller('MyCtrl1', function ($scope) {
    // write Ctrl here

  });
