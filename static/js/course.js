document.addEventListener("DOMContentLoaded", function () {
  const elements = document.getElementsByClassName("cal");
  const url = new URL(document.baseURI);

  const openPrefix = "https://simonrob.github.io/online-ics-feed-viewer/#";
  const googlePrefix = "https://www.google.com/calendar/render?cid=";
  const applePrefix = "";

  for (const el of elements) {
    const pre = el.getElementsByTagName("pre")[0];
    const calPath = pre.textContent;

    const webcalLink = `webcal://${url.host}${calPath}`;

    pre.textContent = webcalLink;
    // Select all text on click
    pre.addEventListener("click", () => {
      const range = document.createRange();
      range.selectNodeContents(pre);
      const sel = window.getSelection();
      sel.removeAllRanges();
      sel.addRange(range);
    });

    const btnCopyUrl = el.getElementsByTagName("button")[0];
    btnCopyUrl.addEventListener("click", () => {
      navigator.clipboard.writeText(pre.textContent);
    });

    const aOpen = el.getElementsByClassName("open")[0];
    aOpen.href =
      openPrefix +
      new URLSearchParams({
        feed: `${url.origin}${calPath}`,
        cors: false,
        title: "Lezioni",
        hideinput: true,
      });

    const addToGoogleBtn = el.getElementsByClassName("google")[0];
    addToGoogleBtn.href = googlePrefix + encodeURIComponent(webcalLink);

    const addToAppleBtn = el.getElementsByClassName("apple")[0];
    addToAppleBtn.href = webcalLink;
  }

  // Utility functions to work with URL params;
  function addSubject(url, subject) {
    if (url.includes("subjects")) {
      url = url.replace(",,", ",");
      url = url.replace("=,", "=");

      let last_char = url.slice(-1);
      if (last_char != "," && last_char != "=") {
        url = url + ",";
      }
      return url + subject;
    } else {
      if (url.includes("?")) {
        return url + "&subjects=" + subject;
      } else {
        return url + "?subjects=" + subject;
      }
    }
  }

  function removeSubject(url, subject) {
    url = url.replace(subject, "");
    url = url.replace(",,", ",");
    url = url.replace("=,", "=");
    return url;
  }

  // Use only the new filter-checkboxes with data-* attributes
  const checkboxes = document.getElementsByClassName
("filter-checkbox");
  for (const ck of checkboxes) {
    ck.checked = false;

    let a = ck.getAttribute("data-anno");
    let c = ck.getAttribute("data-curriculum");
    let o = ck.getAttribute("data-option");
    let innerText = ck.getAttribute("data-innertext");

    ck.addEventListener("click", (event) => {
      // Update both Lezioni and Esami blocks for this anno/curriculum/option
      ["l", "e"].forEach((mode) => {
        let class_name = `${mode}${a}_${c}`;
        let els = document.getElementsByClassName(class_name);
        let plain_string = document.getElementById(class_name).textContent;

        for (const el of els) {
          let res;
          if (ck.checked) {
            res = addSubject(plain_string, o);
          } else {
            res = removeSubject(plain_string, o);
          }

          if (el.nodeName != "A") {
            el.textContent = res;
          } else {
            if (el.classList.contains("open")) {
              el.href =
                openPrefix +
                new URLSearchParams({
                  feed: res.replace("webcal://", "https://"),
                  cors: false,
                  title: mode === "l" ? "Lezioni" : "Esami",
                  hideinput: true,
                });
            } else if (el.classList.contains("google")) {
              el.href = googlePrefix + encodeURIComponent(res);
            } else if (el.classList.contains("apple")) {
              el.href = res;
            }
          }
        }
      });

      // Update the badges only once per block (above Lezioni)
      let badgeContainerClass = `.selected-insegnamenti-badges.l${a}_${c}_badges`;
      let badgeContainer = document.querySelector(badgeContainerClass);
      if (badgeContainer) {
        // Gather all checked checkboxes for this anno/curriculum
        let checked = [];
        for (const otherCk of checkboxes) {
          if (
            otherCk.getAttribute("data-anno") === a &&
            otherCk.getAttribute("data-curriculum") === c &&
            otherCk.checked
          ) {
            checked.push(otherCk.getAttribute("data-innertext"));
          }
        }
        // Render badges
        badgeContainer.innerHTML = checked.length
          ? checked.map(v => `<span class="badge badge-outline badge-sm text-unibo border-[#b5142a]">${v}</span>`).join(" ")
          : "";
      }
    });
  }
});
