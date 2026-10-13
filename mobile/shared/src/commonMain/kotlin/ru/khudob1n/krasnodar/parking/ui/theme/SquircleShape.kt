package ru.khudob1n.krasnodar.parking.ui.theme

import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Outline
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.unit.Density
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.LayoutDirection
import kotlin.math.min

/**
 * Сквиркл - прямоугольник с «непрерывными» углами, как у иконок iOS: кривизна нарастает
 * плавно, без излома на стыке скругления и стороны (--smooth-corners кнопок сайта). Угол
 * занимает 1.52866 радиуса вдоль каждой стороны; кривые - известное приближение
 * UIBezierPath(roundedRect:cornerRadius:).
 */
class SquircleShape(private val radius: Dp) : Shape {
    override fun createOutline(size: Size, layoutDirection: LayoutDirection, density: Density): Outline {
        val w = size.width
        val h = size.height
        val r = min(with(density) { radius.toPx() }, min(w, h) / 2 / 1.52866f)
        val path = Path()
        // Углы по часовой стрелке от верхнего правого; (x, y) угла и направления «вдоль» сторон.
        fun corner(cx: Float, cy: Float, ax: Float, ay: Float, bx: Float, by: Float) {
            // Точка на расстоянии u вдоль входящей стороны (a) и v вдоль исходящей (b) от вершины.
            fun p(u: Float, v: Float) = Pair(cx - ax * u * r + bx * v * r, cy - ay * u * r + by * v * r)
            val (x0, y0) = p(1.52866f, 0f)
            if (path.isEmpty) path.moveTo(x0, y0) else path.lineTo(x0, y0)
            val c1 = p(1.08849f, 0f); val c2 = p(0.86840f, 0f); val e1 = p(0.63149f, 0.07491f)
            path.cubicTo(c1.first, c1.second, c2.first, c2.second, e1.first, e1.second)
            val c3 = p(0.37282f, 0.16906f); val c4 = p(0.16906f, 0.37282f); val e2 = p(0.07491f, 0.63149f)
            path.cubicTo(c3.first, c3.second, c4.first, c4.second, e2.first, e2.second)
            val c5 = p(0f, 0.86840f); val c6 = p(0f, 1.08849f); val e3 = p(0f, 1.52866f)
            path.cubicTo(c5.first, c5.second, c6.first, c6.second, e3.first, e3.second)
        }
        corner(w, 0f, 1f, 0f, 0f, 1f)   // верх -> правая сторона
        corner(w, h, 0f, 1f, -1f, 0f)   // правая -> низ
        corner(0f, h, -1f, 0f, 0f, -1f) // низ -> левая
        corner(0f, 0f, 0f, -1f, 1f, 0f) // левая -> верх
        path.close()
        return Outline.Generic(path)
    }
}
