'use strict';

angular.module('myApp.home', ['ngRoute'])

.config(['$routeProvider', function($routeProvider) {
  $routeProvider.when('/', {
    templateUrl: '/views/home/home.html',
    controller: 'HomeController'
  });
}])

.controller('HomeController', function ($scope, $http, Notification) {
  $scope.getDashboardData = function() {
    $http({
      method: 'GET',
      url: '/data/all-enabled-currencies'
    }).
    success(function (data, status, headers, config) {
      $scope.exchanges = data.data;
      Notification.info("Retrieved latest data");
    }).
    error(function (data, status, headers, config) {
      console.log('error');
    });
  };

  $scope.reloadDashboardWithExchangeCurrency = function (exchange, value) {
    $scope.selectedExchange = exchange;
    $scope.selectedCurrency = value;
  };

  $scope.getDashboardData();

});