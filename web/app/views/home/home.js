'use strict';

angular.module('myApp.home', ['ngRoute'])

.config(['$routeProvider', function($routeProvider) {
  $routeProvider.when('/', {
    templateUrl: '/views/home/home.html',
    controller: 'HomeController'
  });
}])

  .controller('HomeController', function ($scope, $http, Notification, $rootScope) {
    $scope.selected = {};
  $scope.getDashboardData = function() {
    $http({
      method: 'GET',
      url: '/data/all-enabled-currencies'
    }).
    success(function (data, status, headers, config) {
      $scope.exchanges = data.data;
      $scope.reloadDashboardWithExchangeCurrency($scope.exchanges[0], $scope.exchanges[0].exchangeValues[0]);
      Notification.info("Retrieved latest data");
    }).
    error(function (data, status, headers, config) {
      console.log('error');
    });
  };

  $scope.reloadDashboardWithExchangeCurrency = function (exchange, value) {
    $scope.selected.Exchange = exchange;
    $scope.selected.Currency = value;
    $rootScope.$emit('CurrencyChanged', $scope.selected);
    
  };

  $scope.getDashboardData();

});