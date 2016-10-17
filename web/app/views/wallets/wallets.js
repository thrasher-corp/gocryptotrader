'use strict';

angular.module('myApp.wallets', ['ngRoute'])

.config(['$routeProvider', function($routeProvider) {
  $routeProvider.when('/wallets', {
    templateUrl: '/views/wallets/wallets.html',
    controller: 'WalletsController'
  });
}])

.controller('WalletsController', function ($scope, $http, Notification) {
  $scope.getDashboardData = function() {
    $http({
      method: 'GET',
      url: '/data/all-enabled-exchange-account-info'
    }).
    success(function (data, status, headers, config) {
      $scope.wallets = data.data;
      Notification.info("Got your wallet!");
    }).
    error(function (data, status, headers, config) {
      console.log('error');
    });
  };

  $scope.getDashboardData();

});