'use strict';

angular.module('myApp.home', ['ngRoute'])

.config(['$routeProvider', function($routeProvider) {
  $routeProvider.when('/', {
    templateUrl: '/views/home/home.html',
    controller: 'HomeController'
  });
}])

.controller('HomeController', function ($scope, $http) {
  $scope.getDashboardData = function() {
    $http({
      method: 'GET',
      url: '/data/all-enabled-currencies'
    }).
    success(function (data, status, headers, config) {
      $scope.exchanges = data.data;
    }).
    error(function (data, status, headers, config) {
      console.log('error');
    });
  };

  $scope.getDashboardData();

});