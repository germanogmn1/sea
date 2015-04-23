(function () {
  'use strict';

  var query = function (el, selector) {
    if (arguments.length == 1) {
      selector = el;
      el = document;
    }
    return el.querySelectorAll(selector);
  };

  var first = function (el, selector) {
    if (arguments.length == 1) {
      selector = el;
      el = document;
    }
    return el.querySelector(selector);
  };

  var each = function (list, fn) {
    for (var i = 0; i < list.length; i++) {
      fn(list[i]);
    }
  };

  var on = function (el, event, callback) {
    el.addEventListener(event, callback);
  };

  each(query('[data-dropdown]'), function (el) {
    var toggle = first(el, '[data-dropdown-toggle]');
    var options = first(el, '[data-dropdown-options]');

    on(toggle, 'click', function (e) {
      e.preventDefault();
      e.stopPropagation();
      options.hidden = !options.hidden;
    });
  });

  on(document, 'click', function (event) {
    each(query('[data-dropdown] [data-dropdown-options]:not([hidden])'), function (el) {
      el.hidden = true;
    });
  });

})();
