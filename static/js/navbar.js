$(".toggle").click(function () {
    var target = $(this).parents(".nav").find(".menu")
    if (target.hasClass("open")) {
        target.removeClass("open")
    } else {
        target.addClass("open")
    }
})
$(".menu-item").mouseleave(function () {
    $(this).find(".sub-menu").css("display","none")
})
$(".menu-item").mouseenter(function () {
    $(this).find(".sub-menu").css("display","block")
})
let last_known_scroll_position = 0;
window.addEventListener('scroll', function(e) {
if( window.scrollY>last_known_scroll_position){
    $(".nav").css("top","-50px")
    $(".nav").css("opacity","0")
}else{
    $(".nav").css("top","0px")
    $(".nav").css("opacity","1")
}
last_known_scroll_position = window.scrollY;    
});