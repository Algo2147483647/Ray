# ifndef CONSTS_H
# define CONSTS_H

#define EPS std::numeric_limits<float>::epsilon()
thread_local std::mt19937 generator(std::random_device{}());
thread_local std::uniform_real_distribution<> distribution(0, 1);


#endif