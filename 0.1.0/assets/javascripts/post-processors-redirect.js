(function () {
  function redirectPostProcessorsIndex() {
    var path = location.pathname.replace(/\/+$/, "");
    if (!/\/post-processors$/.test(path)) {
      return;
    }
    location.replace(path + "/artifactory/");
  }

  redirectPostProcessorsIndex();

  if (typeof location$ !== "undefined" && location$.subscribe) {
    location$.subscribe(redirectPostProcessorsIndex);
  }
})();
